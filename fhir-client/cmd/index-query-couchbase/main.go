package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/couchbase/gocb/v2"
)

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	ctx := context.Background()

	cbURL := getEnv("COUCHBASE_URL", "couchbase://evt-db")
	user := getEnv("COUCHBASE_USERNAME", "evtechallenge_user")
	pass := getEnv("COUCHBASE_PASSWORD", "password")

	cluster, err := gocb.Connect(cbURL, gocb.ClusterOptions{
		Authenticator:  gocb.PasswordAuthenticator{Username: user, Password: pass},
		TimeoutsConfig: gocb.TimeoutsConfig{QueryTimeout: 30 * time.Second, ConnectTimeout: 30 * time.Second},
	})
	if err != nil {
		panic(fmt.Errorf("connect cluster: %w", err))
	}
	bucket := cluster.Bucket("EvTeChallenge")
	err = bucket.WaitUntilReady(15*time.Second, &gocb.WaitUntilReadyOptions{Context: ctx, ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery}})
	if err != nil {
		panic(fmt.Errorf("bucket not ready: %w", err))
	}

	// 1) Count all resources by collection (will be shown at the end)

	// 2) Get up to 15 encounters, pick the first with both patient and practitioners
	type encRow struct {
		ID              string                 `json:"id"`
		Resource        map[string]interface{} `json:"resource"`
		SubjectPatient  string                 `json:"subjectPatientId"`
		PractitionerIDs []string               `json:"practitionerIds"`
	}

	q1 := "SELECT META(d).id AS id, d AS resource, d.subjectPatientId AS subjectPatientId, d.practitionerIds AS practitionerIds FROM `EvTeChallenge`.`_default`.`encounters` AS d ORDER BY META(d).id LIMIT 5"
	rows, err := cluster.Query(q1, &gocb.QueryOptions{Adhoc: true})
	if err != nil {
		panic(fmt.Errorf("query encounters: %w", err))
	}
	var encs []encRow
	for rows.Next() {
		var r encRow
		if err := rows.Row(&r); err != nil {
			panic(fmt.Errorf("read row: %w", err))
		}
		encs = append(encs, r)
	}
	if err := rows.Err(); err != nil {
		panic(fmt.Errorf("iter rows: %w", err))
	}

	// Debug: Show JSON of first 5 encounters
	fmt.Println("\n=== First 5 Encounters (Debug) ===")
	for i, enc := range encs {
		jsonData, _ := json.MarshalIndent(enc, "", "  ")
		fmt.Printf("Encounter %d:\n%s\n\n", i+1, string(jsonData))
	}

	var picked encRow
	var found bool
	for _, e := range encs {
		pID := e.SubjectPatient
		if pID == "" {
			pID = extractIDFromEncounter(e.Resource, "Patient")
		}
		prIDs := e.PractitionerIDs
		if len(prIDs) == 0 {
			prIDs = extractPractitionerIDsFromEncounter(e.Resource)
		}
		if pID != "" && len(prIDs) > 0 {
			picked = e
			picked.SubjectPatient = pID
			picked.PractitionerIDs = prIDs
			found = true
			break
		}
	}
	if !found {
		fmt.Println("No encounter with both patient and practitioners found in first 5.")
		return
	}
	fmt.Printf("Picked encounter key: %s\n", picked.ID)

	// 3) Re-query this encounter by identifier (key)
	q2 := "SELECT META(d).id AS id FROM `EvTeChallenge`.`_default`.`encounters` AS d USE KEYS $key"
	rows2, err := cluster.Query(q2, &gocb.QueryOptions{NamedParameters: map[string]interface{}{"key": picked.ID}})
	if err != nil {
		panic(fmt.Errorf("requery by key: %w", err))
	}
	var foundKey string
	for rows2.Next() {
		var row struct {
			ID string `json:"id"`
		}
		if err := rows2.Row(&row); err != nil {
			panic(fmt.Errorf("row: %w", err))
		}
		foundKey = row.ID
	}
	if err := rows2.Err(); err != nil {
		panic(fmt.Errorf("iter: %w", err))
	}
	fmt.Printf("Re-query returned key: %s (match=%v)\n", foundKey, foundKey == picked.ID)

	// 4) Extract patient/practitioner IDs from the encounter (already computed)
	patientID := picked.SubjectPatient
	practitionerIDs := picked.PractitionerIDs
	fmt.Printf("Encounter patientID: %s\n", patientID)
	fmt.Printf("Encounter practitionerIDs: %v\n", practitionerIDs)

	// 5) Test query by these IDs using collections
	if patientID != "" {
		ok := existsByCollectionAndID(cluster, "patients", patientID)
		fmt.Printf("Patient %s exists by N1QL: %v\n", patientID, ok)
	}
	for _, pid := range practitionerIDs {
		ok := existsByCollectionAndID(cluster, "practitioners", pid)
		fmt.Printf("Practitioner %s exists by N1QL: %v\n", pid, ok)
	}

	// 6) List all Patients and verify membership
	allPatients := listIDsByCollection(cluster, "patients")
	if patientID != "" {
		fmt.Printf("Patient %s present in full list: %v\n", patientID, contains(allPatients, patientID))
	}
	// 7) Same for Practitioners
	allPractitioners := listIDsByCollection(cluster, "practitioners")
	for _, pid := range practitionerIDs {
		fmt.Printf("Practitioner %s present in full list: %v\n", pid, contains(allPractitioners, pid))
	}

	// 8) Demonstrate collection-based JOIN query
	fmt.Println("\n=== Collection-based JOIN Demo ===")
	if patientID != "" && len(practitionerIDs) > 0 {
		joinQuery := `
			SELECT 
				e.id as encounter_id,
				e.subjectPatientId,
				p.id as patient_id,
				pr.id as practitioner_id
			FROM EvTeChallenge._default.encounters e
			LEFT JOIN EvTeChallenge._default.patients p ON e.subjectPatientId = p.id
			LEFT JOIN EvTeChallenge._default.practitioners pr ON pr.id IN e.practitionerIds
			WHERE e.id = $encounter_id
			LIMIT 1`

		rows, err := cluster.Query(joinQuery, &gocb.QueryOptions{
			NamedParameters: map[string]interface{}{"encounter_id": picked.ID},
		})
		if err != nil {
			fmt.Printf("Join query error: %v\n", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var result struct {
					EncounterID    string `json:"encounter_id"`
					SubjectPatient string `json:"subjectPatientId"`
					PatientID      string `json:"patient_id"`
					PractitionerID string `json:"practitioner_id"`
				}
				if err := rows.Row(&result); err == nil {
					fmt.Printf("JOIN Result: Encounter=%s, Patient=%s, Practitioner=%s\n",
						result.EncounterID, result.PatientID, result.PractitionerID)
				}
			}
		}
	}

	// Final counts and summary
	fmt.Println("\n=== Final Summary ===")
	fmt.Printf("Total Encounters: %d\n", countByCollection(cluster, "encounters"))
	fmt.Printf("Total Patients: %d\n", countByCollection(cluster, "patients"))
	fmt.Printf("Total Practitioners: %d\n", countByCollection(cluster, "practitioners"))

	// Count valid patient and practitioner references found in encounters
	validPatientRefs := countValidReferencesInEncounters(cluster, "Patient")
	validPractitionerRefs := countValidReferencesInEncounters(cluster, "Practitioner")
	fmt.Printf("Valid Patient references found in encounters: %d\n", validPatientRefs)
	fmt.Printf("Valid Practitioner references found in encounters: %d\n", validPractitionerRefs)

	// Debug: Show some actual IDs from collections
	fmt.Println("\n=== Debug: Sample IDs from Collections ===")
	allPatientIDs := listIDsByCollection(cluster, "patients")
	allPractitionerIDs := listIDsByCollection(cluster, "practitioners")

	fmt.Printf("First 5 Patient IDs: %v\n", allPatientIDs[:min(5, len(allPatientIDs))])
	fmt.Printf("First 5 Practitioner IDs: %v\n", allPractitionerIDs[:min(5, len(allPractitionerIDs))])
}

// countByCollection counts the number of resources by collection
func countByCollection(cluster *gocb.Cluster, collection string) int {
	q := fmt.Sprintf("SELECT COUNT(*) AS cnt FROM `EvTeChallenge`.`_default`.`%s`", collection)
	r, err := cluster.Query(q, &gocb.QueryOptions{})
	if err != nil {
		fmt.Printf("countByCollection query error: %v\n", err)
		return 0
	}
	defer r.Close()

	var count int
	for r.Next() {
		var row struct {
			Cnt int `json:"cnt"`
		}
		if err := r.Row(&row); err == nil {
			count = row.Cnt
		}
	}
	return count
}

// existsByCollectionAndID checks if a resource exists by collection and ID
func existsByCollectionAndID(cluster *gocb.Cluster, collection, id string) bool {
	q := fmt.Sprintf("SELECT 1 FROM `EvTeChallenge`.`_default`.`%s` AS d WHERE d.`id`=$id LIMIT 1", collection)
	r, err := cluster.Query(q, &gocb.QueryOptions{NamedParameters: map[string]interface{}{"id": id}})
	if err != nil {
		return false
	}
	defer r.Close()
	return r.Next()
}

// listIDsByCollection lists all IDs by collection
func listIDsByCollection(cluster *gocb.Cluster, collection string) []string {
	q := fmt.Sprintf("SELECT d.`id` AS id FROM `EvTeChallenge`.`_default`.`%s` AS d", collection)
	r, err := cluster.Query(q, &gocb.QueryOptions{})
	if err != nil {
		return nil
	}
	defer r.Close()
	var ids []string
	for r.Next() {
		var row struct {
			ID string `json:"id"`
		}
		if err := r.Row(&row); err == nil && row.ID != "" {
			ids = append(ids, row.ID)
		}
	}
	return ids
}

// countValidReferencesInEncounters counts how many encounters have valid references to the specified resource type
func countValidReferencesInEncounters(cluster *gocb.Cluster, resourceType string) int {
	var fieldName string
	if resourceType == "Patient" {
		fieldName = "subjectPatientId"
	} else if resourceType == "Practitioner" {
		fieldName = "practitionerIds"
	} else {
		return 0
	}

	var query string
	if resourceType == "Patient" {
		query = fmt.Sprintf("SELECT COUNT(*) AS cnt FROM `EvTeChallenge`.`_default`.`encounters` WHERE %s IS NOT NULL AND %s != ''", fieldName, fieldName)
	} else {
		query = fmt.Sprintf("SELECT COUNT(*) AS cnt FROM `EvTeChallenge`.`_default`.`encounters` WHERE %s IS NOT NULL AND ARRAY_LENGTH(%s) > 0", fieldName, fieldName)
	}

	r, err := cluster.Query(query, &gocb.QueryOptions{})
	if err != nil {
		return 0
	}
	defer r.Close()

	var count int
	for r.Next() {
		var row struct {
			Cnt int `json:"cnt"`
		}
		if err := r.Row(&row); err == nil {
			count = row.Cnt
		}
	}
	return count
}

// contains checks if an array contains a value
func contains(arr []string, v string) bool {
	for _, x := range arr {
		if x == v {
			return true
		}
	}
	return false
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractIDFromEncounter extracts the ID from an encounter
func extractIDFromEncounter(enc map[string]interface{}, resourceType string) string {
	// subject.reference: "Patient/<id>"
	if resourceType != "Patient" {
		return ""
	}

	subj, ok := enc["subject"].(map[string]interface{})
	if !ok {
		return ""
	}

	ref, ok := subj["reference"].(string)
	if !ok {
		return ""
	}

	id := extractIDFromReference(ref, "Patient")
	if id != "" {
		return id
	}

	return ""
}

// extractPractitionerIDsFromEncounter extracts the practitioner IDs from an encounter
func extractPractitionerIDsFromEncounter(enc map[string]interface{}) []string {
	var ids []string

	parts, ok := enc["participant"].([]interface{})
	if !ok {
		return ids
	}

	for _, p := range parts {
		pm, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		ind, ok := pm["individual"].(map[string]interface{})
		if !ok {
			continue
		}

		ref, ok := ind["reference"].(string)
		if !ok {
			continue
		}

		id := extractIDFromReference(ref, "Practitioner")
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids
}

// extractIDFromReference extracts the ID from a reference
func extractIDFromReference(reference, resourceType string) string {
	if strings.Contains(reference, "/") {
		parts := strings.Split(reference, "/")
		if len(parts) == 2 {
			if parts[0] == resourceType {
				return parts[1]
			}
		}
	}
	return ""
}
