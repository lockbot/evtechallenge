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
