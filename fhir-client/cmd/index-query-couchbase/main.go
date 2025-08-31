package main

import (
	"context"
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

	cbURL := getEnv("COUCHBASE_URL", "couchbase://evtechallenge-db")
	user := getEnv("COUCHBASE_USERNAME", "evtechallenge_user")
	pass := getEnv("COUCHBASE_PASSWORD", "password")

	cluster, err := gocb.Connect(cbURL, gocb.ClusterOptions{
		Authenticator:  gocb.PasswordAuthenticator{Username: user, Password: pass},
		TimeoutsConfig: gocb.TimeoutsConfig{QueryTimeout: 30 * time.Second, ConnectTimeout: 30 * time.Second},
	})
	if err != nil {
		panic(fmt.Errorf("connect cluster: %w", err))
	}
	bucket := cluster.Bucket("evtechallenge")
	if err := bucket.WaitUntilReady(60*time.Second, &gocb.WaitUntilReadyOptions{Context: ctx, ServiceTypes: []gocb.ServiceType{gocb.ServiceTypeKeyValue, gocb.ServiceTypeQuery}}); err != nil {
		panic(fmt.Errorf("bucket not ready: %w", err))
	}

	// 1) Get up to 15 encounters, pick the first with both patient and practitioners
	type encRow struct {
		ID              string                 `json:"id"`
		Resource        map[string]interface{} `json:"resource"`
		SubjectPatient  string                 `json:"subjectPatientId"`
		PractitionerIDs []string               `json:"practitionerIds"`
	}

	q1 := "SELECT META(d).id AS id, d AS resource, d.subjectPatientId AS subjectPatientId, d.practitionerIds AS practitionerIds FROM `evtechallenge` AS d WHERE d.`resourceType` = 'Encounter' LIMIT 15"
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
		fmt.Println("No encounter with both patient and practitioners found in first 15.")
		return
	}
	fmt.Printf("Picked encounter key: %s\n", picked.ID)

	// 2) Re-query this encounter by identifier (key)
	q2 := "SELECT META(d).id AS id FROM `evtechallenge` AS d USE KEYS $key"
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

	// 3) Extract patient/practitioner IDs from the encounter (already computed)
	patientID := picked.SubjectPatient
	practitionerIDs := picked.PractitionerIDs
	fmt.Printf("Encounter patientID: %s\n", patientID)
	fmt.Printf("Encounter practitionerIDs: %v\n", practitionerIDs)

	// 4) Test query by these IDs (N1QL on resourceType+id)
	if patientID != "" {
		ok := existsByTypeAndID(cluster, "Patient", patientID)
		fmt.Printf("Patient %s exists by N1QL: %v\n", patientID, ok)
	}
	for _, pid := range practitionerIDs {
		ok := existsByTypeAndID(cluster, "Practitioner", pid)
		fmt.Printf("Practitioner %s exists by N1QL: %v\n", pid, ok)
	}

	// 5) List all Patients and verify membership
	allPatients := listIDsByType(cluster, "Patient")
	if patientID != "" {
		fmt.Printf("Patient %s present in full list: %v\n", patientID, contains(allPatients, patientID))
	}
	// 6) Same for Practitioners
	allPractitioners := listIDsByType(cluster, "Practitioner")
	for _, pid := range practitionerIDs {
		fmt.Printf("Practitioner %s present in full list: %v\n", pid, contains(allPractitioners, pid))
	}
}

func existsByTypeAndID(cluster *gocb.Cluster, rt, id string) bool {
	q := "SELECT 1 FROM `evtechallenge` AS d WHERE d.`resourceType`=$rt AND d.`id`=$id LIMIT 1"
	r, err := cluster.Query(q, &gocb.QueryOptions{NamedParameters: map[string]interface{}{"rt": rt, "id": id}})
	if err != nil {
		return false
	}
	defer r.Close()
	return r.Next()
}

func listIDsByType(cluster *gocb.Cluster, rt string) []string {
	q := "SELECT d.`id` AS id FROM `evtechallenge` AS d WHERE d.`resourceType`=$rt"
	r, err := cluster.Query(q, &gocb.QueryOptions{NamedParameters: map[string]interface{}{"rt": rt}})
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

func contains(arr []string, v string) bool {
	for _, x := range arr {
		if x == v {
			return true
		}
	}
	return false
}

func extractIDFromEncounter(enc map[string]interface{}, resourceType string) string {
	// subject.reference: "Patient/<id>"
	if resourceType == "Patient" {
		if subj, ok := enc["subject"].(map[string]interface{}); ok {
			if ref, ok := subj["reference"].(string); ok {
				id := extractIDFromReference(ref, "Patient")
				if id != "" {
					return id
				}
			}
		}
	}
	return ""
}

func extractPractitionerIDsFromEncounter(enc map[string]interface{}) []string {
	var ids []string
	if parts, ok := enc["participant"].([]interface{}); ok {
		for _, p := range parts {
			if pm, ok := p.(map[string]interface{}); ok {
				if ind, ok := pm["individual"].(map[string]interface{}); ok {
					if ref, ok := ind["reference"].(string); ok {
						id := extractIDFromReference(ref, "Practitioner")
						if id != "" {
							ids = append(ids, id)
						}
					}
				}
			}
		}
	}
	return ids
}

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
