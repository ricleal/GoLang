#!/bin/bash

# This script performs a series of load tests using vegeta
# to test a load shedding algorithm designed to handle
# traffic above 100 requests per second.

# Define the target URL for the test
# IMPORTANT: Replace this with the actual URL you want to test.
TARGET_URL="http://localhost:8080"

# Define the duration of each attack in seconds
DURATION="10s"

# Define the rates to test, in requests per second.
# We start below the 100 rps threshold, hit it, and then go above.
# The 'T' in vegeta format means 'requests per second'.
RATES=("50/1s" "75/1s" "100/1s" "125/1s" "150/1s" "200/1s")

# Clean up any previous report/plot files
rm -f report_*.txt
rm -f plot_*.html
rm -f temp_results_*.bin

# Define the target request format for vegeta
echo "GET ${TARGET_URL}" > targets.txt

echo "Starting load tests..."
echo "-------------------------------------"

# Loop through each defined rate and run an attack
for RATE in "${RATES[@]}"; do
    # Create a unique temporary file for each attack's results
    TEMP_FILE="temp_results_${RATE//\//_}.bin"
    
    echo "Attacking with a rate of ${RATE} for ${DURATION}..."
    
    # Run the vegeta attack and save the binary output to a temporary file
    vegeta attack \
        -targets=targets.txt \
        -rate="${RATE}" \
        -duration="${DURATION}" \
        -workers=50 \
        -keepalive \
        -output="${TEMP_FILE}"
        
    echo "Attack completed for rate ${RATE}."
    
    # Generate a summary report from the temporary binary results file
    echo "Generating report for rate ${RATE}..."
    cat "${TEMP_FILE}" | vegeta report > "report_${RATE//\//_}.txt"
    
    # Generate a visual HTML plot from the temporary binary results file
    echo "Generating plot for rate ${RATE}..."
    cat "${TEMP_FILE}" | vegeta plot > "plot_${RATE//\//_}.html"
    
    echo "-------------------------------------"
    
    # Wait for a few seconds to allow the system to cool down before the next test
    sleep 1
done

# Clean up the temporary binary files
rm -f temp_results_*.bin

echo "-------------------------------------"
echo "Load test complete."
echo "Reports and plots for each rate have been saved to individual files:"
echo " - report_*.txt"
echo " - plot_*.html"
echo "Check these files to analyze the performance at each load level."
