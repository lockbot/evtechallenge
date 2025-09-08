// PRE-REQUEST SCRIPT: JWT Token Renewal
// Check if token is expired or missing
const currentToken = pm.environment.get("access_token");
const tokenExpiresAt = pm.environment.get("token_expires_at");
const refreshToken = pm.environment.get("refresh_token");

// If token is expired or missing, try to refresh it
if (!currentToken || !tokenExpiresAt || Date.now() > tokenExpiresAt) {
    console.log("Token expired or missing, attempting refresh...");
    
    if (refreshToken) {
        // Try refresh token first
        pm.sendRequest({
            url: pm.environment.get("keycloak_url") + "/realms/" + pm.environment.get("realm") + "/protocol/openid-connect/token",
            method: "POST",
            header: {
                "Content-Type": "application/x-www-form-urlencoded"
            },
            body: {
                mode: "urlencoded",
                urlencoded: [
                    { key: "grant_type", value: "refresh_token" },
                    { key: "client_id", value: pm.environment.get("client_id") },
                    { key: "client_secret", value: pm.environment.get("client_secret") },
                    { key: "refresh_token", value: refreshToken }
                ]
            }
        }, function (err, response) {
            if (err || !response || response.code !== 200) {
                console.log("Refresh token failed, falling back to password grant...");
                if (err) {
                    console.error("Refresh token error:", err);
                } else if (response) {
                    console.error("Refresh token response error:", response.code, response.text());
                }
                // Fallback to password grant
                pm.sendRequest({
                    url: pm.environment.get("keycloak_url") + "/realms/" + pm.environment.get("realm") + "/protocol/openid-connect/token",
                    method: "POST",
                    header: {
                        "Content-Type": "application/x-www-form-urlencoded"
                    },
                    body: {
                        mode: "urlencoded",
                        urlencoded: [
                            { key: "grant_type", value: "password" },
                            { key: "client_id", value: pm.environment.get("client_id") },
                            { key: "client_secret", value: pm.environment.get("client_secret") },
                            { key: "username", value: pm.environment.get("username") },
                            { key: "password", value: pm.environment.get("password") }
                        ]
                    }
                }, function (err2, response2) {
                    if (err2 || !response2 || response2.code !== 200) {
                        if (err2) {
                            console.error("Password grant error:", err2);
                        } else if (response2) {
                            console.error("Password grant response error:", response2.code, response2.text());
                        } else {
                            console.error("Password grant failed: no response");
                        }
                        return;
                    }
                    
                    const tokenData = response2.json();
                    pm.environment.set("access_token", tokenData.access_token);
                    pm.environment.set("refresh_token", tokenData.refresh_token);
                    pm.environment.set("token_expires_at", Date.now() + (tokenData.expires_in * 1000));
                    console.log("New tokens obtained via password grant");
                });
            } else {
                const tokenData = response.json();
                pm.environment.set("access_token", tokenData.access_token);
                pm.environment.set("refresh_token", tokenData.refresh_token);
                pm.environment.set("token_expires_at", Date.now() + (tokenData.expires_in * 1000));
                console.log("Tokens refreshed successfully");
            }
        });
    } else {
        console.error("No refresh token available and current token is expired!");
    }
}

// Set Authorization header for the current request
pm.request.headers.add({
    key: "Authorization",
    value: "Bearer " + pm.environment.get("access_token")
});

// Set X-Tenant-ID header
pm.request.headers.add({
    key: "X-Tenant-ID",
    value: pm.environment.get("username")
});

// POST-REQUEST SCRIPT: Convert legacy URLs to new format
pm.test("Status code is 200", function () {
    pm.response.to.have.status(200);
});

// Parse the JSON response
let response = pm.response.json();

// Convert to string so we can regex match easily
let responseStr = JSON.stringify(response);

// Find all occurrences of /Encounter/[\w\d-]+
let matches = responseStr.match(/Encounter\/[\w\d-]+/g) || [];

// Sort alphabetically
matches.sort();

// Count total occurrences
let totalCount = matches.length;

// Count unique occurrences and frequencies
let freqMap = {};
matches.forEach(m => {
    freqMap[m] = (freqMap[m] || 0) + 1;
});

// Build result JSON
let result = {
    totalOccurrences: totalCount,
    uniqueCount: Object.keys(freqMap).length,
    uniqueOccurrences: Object.keys(freqMap).map(k => ({
        encounter: k,
        count: freqMap[k]
    }))
};

// Print to Postman Console
console.log("Result:", result);

// Make it available as a fake "test" output
pm.test("Custom Output", function () {
    pm.expect(true).to.be.true;
    console.log("Final JSON:", result);
});

// Save to environment variable if you want to inspect
pm.environment.set("encounter_summary", JSON.stringify(result, null, 2));

// URL CONVERSION: Convert legacy URLs to new format
// This converts /legacy/... URLs to /api/{tenant}/... format
const currentUrl = pm.request.url.toString();
const tenantId = pm.environment.get("username") || "tenant1";

if (currentUrl.includes("/legacy/")) {
    const newUrl = currentUrl.replace("/legacy/", `/api/${tenantId}/`);
    console.log("URL converted from legacy format:");
    console.log("  Old:", currentUrl);
    console.log("  New:", newUrl);
    
    // Store the converted URL for reference
    pm.environment.set("converted_url", newUrl);
    
    // Note: This is just for logging - the actual request has already been made
    // To use the new URL, you would need to make a new request with the converted URL
    pm.test("URL Conversion Info", function () {
        pm.expect(true).to.be.true;
        console.log("For future requests, use the new URL format:", newUrl);
    });
}
