# New Multi-Tenancy Architecture

## 🏗️ **Overview**

This document describes the new simplified multi-tenancy architecture that replaces the complex review system with a cleaner, more logical approach.

## 🔄 **Data Flow**

### **1. FHIR Client (One-time Setup)**
```
FHIR Client Start → Set "ready=false" in _system/ingestion_status
                → Populate DefaultCollection with FHIR data
                → Set "ready=true" in _system/ingestion_status
                → Exit
```

### **2. API REST Service**
```
API Start → Wait for "ready=true" (check every 5 seconds)
         → Initialize TenantGoroutineManager
         → Serve requests (only hello, all-good, metrics, health, warm-up-tenant)
```

### **3. Tenant Warm-Up Flow** 🚀
```
B2B Client → POST /warm-up-tenant → Start tenant goroutine → Initialize collection → Mark ready → Return 200 OK
```

### **4. Tenant Request Flow** 🚀
```
FHIR Request → Check if tenant is warm → If cold: Return 503 "call warm-up-tenant"
            → If warm: Record activity → Serve from tenant collection
            → After 30 min inactivity: Goroutine stops → Route returns 503
```

### **5. Goroutine Lifecycle** 🚀
```
Warm-Up → Initialize Collection → Mark Ready → Wait for Activity → 30min Timeout → Go Cold → Stop Goroutine
```

## 📁 **Collection Structure**

- **`_default`** = Master FHIR data (populated once by FHIR client)
- **`tenant_<tenant_id>`** = Each tenant's isolated data (copied from _default)

## 📄 **Document Structure**

### **Ingestion Status Document**
```json
{
  "_id": "_system/ingestion_status",
  "ready": true,
  "startedAt": "2024-01-01T10:00:00Z",
  "completedAt": "2024-01-01T10:05:00Z",
  "message": "FHIR ingestion completed successfully"
}
```

### **FHIR Resources (with review field)**
```json
{
  "_id": "Encounter/123",
  "resourceType": "Encounter",
  "subject": { "reference": "Patient/456" },
  "reviewed": false,
  "reviewTime": null
}
```

## 🔧 **Key Components**

### **FHIR Client (fhir-client/)**
- **`CheckAndSetIngestionStatus()`** - Checks if ingestion already completed, exits gracefully if so
- **`SetIngestionComplete()`** - Marks ingestion as complete when finished
- **Automatic exit** - If finds `ready=true`, exits gracefully instead of re-ingesting
- **Status persistence** - Stores status in `_system/ingestion_status` document

### **API REST (api-rest/)**
- **`TenantCollectionManager`** - Manages tenant collections and waits for FHIR
- **`WaitForFHIRIngestion()`** - Waits for FHIR to complete (checks every 5 seconds)
- **`EnsureTenantCollection()`** - Creates tenant collections on-demand
- **`GetTenantCollection()`** - Returns tenant-specific collection
- **`TenantGoroutineManager`** - 🚀 **NEW!** Manages tenant goroutines with warm-up/cool-down cycles
- **`TenantWarmthMiddleware`** - 🚀 **NEW!** Protects FHIR routes, only allows warm tenants
- **`/warm-up-tenant`** - 🚀 **NEW!** Endpoint for B2B clients to start tenant goroutines

### **Simplified Review System**
- No more complex review maps or separate review documents
- Just `reviewed: boolean` field on each entity
- `reviewTime: string` when reviewed
- Direct updates to entity documents
- **No more `GetReviewInfo()` calls** - review info comes directly from document fields
- **Eliminates N+1 query problem** - no more looping through resources to fetch review status

## 🚀 **Benefits**

1. **Cleaner Architecture** - No complex review document management
2. **Better Performance** - Direct entity queries instead of map lookups
3. **Easier Maintenance** - Simpler code, fewer moving parts
4. **True Multi-Tenancy** - Each tenant gets their own collection
5. **Automatic Setup** - Tenant collections created on-demand

## 📝 **Implementation Notes**

- FHIR client must set ingestion status before starting
- API REST waits for FHIR completion before serving requests
- Tenant collections are created automatically on first request
- Review system is now just a boolean field + timestamp
- All queries use tenant collections, not DefaultCollection

## 🔍 **Next Steps**

1. ✅ **Integrate ingestion status** into actual FHIR client
2. ✅ **Simplify review system** - Remove GetReviewInfo calls, use document fields directly
3. **Implement proper collection copying** (currently simplified)
4. ✅ **Update all database queries** to use tenant collections
5. **Test with multiple tenants**

## ⚠️ **Important Changes**

- **Removed**: Complex review document system
- **Removed**: ReviewInfo struct and related types
- **Added**: Ingestion status management
- **Added**: Tenant collection management
- **Simplified**: Review system to just boolean + timestamp
