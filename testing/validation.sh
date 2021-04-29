#!/bin/bash
ORKESTRA_RESOURCE_COUNT=5
LOG_FILE="OrkestraValidation.log"
OUTPUT_TO_LOG=0
g_successCount=0
g_failureCount=0

while getopts "f" flag; do
    case "${flag}" in
        f)  OUTPUT_TO_LOG=1;;
    esac
done

function outputMessage {
    if [ "$OUTPUT_TO_LOG" -eq 1 ]; then
        echo $1 &>> $LOG_FILE
    else
        echo $1
    fi
}

function testSuiteMessage {
    if [ "$1" == "TEST_PASS" ]; then
        outputMessage "SUCCESS: $2" 
        ((g_successCount++))
    elif [ "$1" == "TEST_FAIL" ]; then
        outputMessage "FAIL: $2" 
        ((g_failureCount++))
    elif [ "$1" == "LOG" ]; then
        outputMessage "LOG: $2"
    fi
}

function summary {
    outputMessage "Success Cases: $g_successCount"
    outputMessage "Failure Cases: $g_failureCount"
}

function resetLogFile {
    > $LOG_FILE
}

function validateOrkestraDeployment {
    resources=$(kubectl get pods --namespace orkestra 2>> $LOG_FILE | grep -i -c running)
    if [ $resources -ne $ORKESTRA_RESOURCE_COUNT ]; then
        testSuiteMessage "TEST_FAIL" "No running orkestra resources. Currently $resources running resources. Expected $ORKESTRA_RESOURCE_COUNT"
    else
        testSuiteMessage "TEST_PASS" "orkestra resources are running"
    fi

    orkestraStatus=$(helm status orkestra -n orkestra 2>> $LOG_FILE | grep -c deployed)
    if [ $orkestraStatus -eq 1 ]; then
        testSuiteMessage "TEST_PASS" "orkestra deployed successfully"
    else
        testSuiteMessage "TEST_FAIL" "orkestra not deployed"
    fi
}

function validateBookInfoDeployment {
    ambassadorStatus=$(helm status ambassador -n ambassador 2>> $LOG_FILE | grep -c deployed)
    if [ $ambassadorStatus -eq 1 ]; then
        testSuiteMessage "TEST_PASS" "ambassador deployed successfully"
    else
        testSuiteMessage "TEST_FAIL" "ambassador not deployed"
    fi

    bookinfoReleaseNames=("details" "productpage" "ratings" "reviews" "bookinfo")

    for var in "${bookinfoReleaseNames[@]}"
    do  
        deployedStatus=$(helm status $var -n bookinfo 2>> $LOG_FILE | grep -c deployed)
        if [ $deployedStatus -eq 1 ]; then
            testSuiteMessage "TEST_PASS" "$var deployed successfully"
        else
            testSuiteMessage "TEST_FAIL" "$var not deployed"
        fi
    done
}

function validateArgoWorkflow {
    bookinfoStatus=$(curl -s --request GET --url http://localhost:2746/api/v1/workflows/orkestra/bookinfo | grep -c "not found")
    if [ "$bookinfoStatus" -eq 1 ]; then
        testSuiteMessage "TEST_FAIL" "No argo workflow found for bookinfo"
    else
        argoNodes=($(curl -s --request GET --url http://localhost:2746/api/v1/workflows/orkestra/bookinfo | jq -c '.status.nodes[] | {id: .id, name: .name, displayName: .displayName, phase: .phase}'))

        requiredNodes=(
            "bookinfo" 
            "bookinfo.bookinfo.ratings" 
            "bookinfo.ambassador" 
            "bookinfo.bookinfo.details"
            "bookinfo.bookinfo.productpage"
            "bookinfo.ambassador.ambassador"
            "bookinfo.bookinfo.reviews"
            "bookinfo.bookinfo.bookinfo"
            "bookinfo.bookinfo"
        )

        for node in "${requiredNodes[@]}"
        do
            status=$(curl -s --request GET --url http://localhost:2746/api/v1/workflows/orkestra/bookinfo | jq --arg node "$node" -r '.status.nodes[] | select(.name==$node) | .phase')
            if [ "$status" == "Succeeded" ]; then
                testSuiteMessage "TEST_PASS" "argo node: $node has succeeded"
            else
                testSuiteMessage "TEST_FAIL" "$node status: $status, Expected Succeeded"
            fi
        done
    fi
}

function validateApplicationGroup {
    applicationGroupJson=$(kubectl get applicationgroup bookinfo -o json | jq '.status')

    groupReason=$(echo "$applicationGroupJson" | jq -r '.conditions[0].reason')
    groupType=$(echo "$applicationGroupJson" | jq -r '.conditions[0].type')

    if [ "$groupReason" == "Succeeded" ] && [ "$groupType" == "Ready" ]; then
        testSuiteMessage "TEST_PASS" "ApplicationGroup status correct"
    else
        testSuiteMessage "TEST_FAIL" "ApplicationGroup status. Expected (Succeeded, Ready). Got ($groupReason, $groupType)"
    fi

    applicationsJson=$(echo "$applicationGroupJson" | jq '.status')

    ambassadorReason=$(echo "$applicationsJson" | jq -r '.[0].conditions[0].reason')
    ambassadorType=$(echo "$applicationsJson" | jq -r '.[0].conditions[0].type')
    if [ "$ambassadorReason" == "Succeeded" ] && [ "$ambassadorType" == "Ready" ]; then
        testSuiteMessage "TEST_PASS" "Ambassador status correct"
    else
        testSuiteMessage "TEST_FAIL" "Ambassador status. Expected (Succeeded, Ready). Got ($ambassadorReason, $ambassadorType)"
    fi

    bookInfoReason=$(echo "$applicationsJson" | jq -r '.[1].conditions[0].reason')
    bookInfoType=$(echo "$applicationsJson" | jq -r '.[1].conditions[0].type')
    if [ "$bookInfoReason" == "Succeeded" ] && [ "$bookInfoType" == "Ready" ]; then
        testSuiteMessage "TEST_PASS" "BookInfo status correct"
    else
        testSuiteMessage "TEST_FAIL" "BookInfo status. Expected (Succeeded, Ready). Got ($bookInfoReason, $bookInfoType)"
    fi

    subcharts=("details" "productpage" "ratings" "reviews")
    
    for chart in "${subcharts[@]}"
    do
        applicationReason=$(echo "$applicationsJson" | jq -r --arg c "$chart" '.[1].subcharts[$c].conditions[0].reason')
        applicationType=$(echo "$applicationsJson" | jq -r --arg c "$chart" '.[1].subcharts[$c].conditions[0].type')
        if [ "$applicationReason" == "Succeeded" ] && [ "$applicationType" == "Ready" ]; then
            testSuiteMessage "TEST_PASS" "$chart status correct"
        else
            testSuiteMessage "TEST_FAIL" "$chart status. Expected (Succeeded, Ready). Got ($applicationReason, $applicationType)"
        fi
    done

}

function runValidation {
    if [ "$OUTPUT_TO_LOG" -eq 1 ]; then
        resetLogFile
    fi
    echo Running Validation
    validateOrkestraDeployment
    validateBookInfoDeployment
    validateArgoWorkflow
    validateApplicationGroup
    summary
    echo DONE
}

runValidation
