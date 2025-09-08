// JWT Token Renewal Script for Postman
// This script automatically renews JWT tokens using refresh tokens
// Add this as a pre-request script to any request that needs authentication

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
