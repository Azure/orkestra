#!/bin/bash
ORKESTRA_RESOURCE_COUNT=6
AMBASSADOR_VERSION="6.6.0"
BAD_AMBASSADOR_VERSION="100.0.0"
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
    echo $applicationGroupJson
    groupCondition=$(echo "$applicationGroupJson" | jq -r '.conditions[] | select(.reason=="Succeeded") | select(.type=="Ready")')
    if [ -n "$groupCondition" ]; then
        testSuiteMessage "TEST_PASS" "ApplicationGroup status correct"
    else
        testSuiteMessage "TEST_FAIL" "ApplicationGroup status expected: (Succeeded, Ready)"
    fi

    applicationsJson=$(echo "$applicationGroupJson" | jq '.status')
    ambassadorReason=$(echo "$applicationsJson" | jq -r '.[0].conditions[] | select(.reason=="InstallSucceeded")')
    if [ -n "$ambassadorReason" ]; then
        testSuiteMessage "TEST_PASS" "Ambassador status correct"
    else
        testSuiteMessage "TEST_FAIL" "Ambassador status expected: InstallSucceeded"
    fi

    bookInfoReason=$(echo "$applicationsJson" | jq -r '.[1].conditions[] | select(.reason=="InstallSucceeded")')
    if [ -n "$bookInfoReason" ]; then
        testSuiteMessage "TEST_PASS" "BookInfo status correct"
    else
        testSuiteMessage "TEST_FAIL" "BookInfo status expected: InstallSucceeded"
    fi

    subcharts=("details" "productpage" "ratings" "reviews")
    for chart in "${subcharts[@]}"
    do
        applicationReason=$(echo "$applicationsJson" | jq -r --arg c "$chart" '.[1].subcharts[$c].conditions[] | select(.reason=="InstallSucceeded")')
        if [ -n "$applicationReason" ]; then
            testSuiteMessage "TEST_PASS" "$chart status correct"
        else
            testSuiteMessage "TEST_FAIL" "$chart status expected: InstallSucceeded"
        fi
    done

}

function applyFailureOnExistingDeployment {
    kubectl get deployments.apps orkestra -n orkestra -o json | jq '.spec.template.spec.containers[].args += ["--disable-remediation"]' | kubectl replace -f -
    kubectl get applicationgroup bookinfo -o json | jq --arg v "$BAD_AMBASSADOR_VERSION" '.spec.applications[0].spec.chart.version = $v' | kubectl replace -f -
}

function deployFailure {
    kubectl delete applicationgroup bookinfo
    sed "s/${AMBASSADOR_VERSION}/${BAD_AMBASSADOR_VERSION}/g" ./examples/simple/bookinfo.yaml | kubectl apply -f -
    sleep 5
}

function validateFailedApplicationGroup {
    applicationGroupJson=$(kubectl get applicationgroup bookinfo -o json | jq '.status')
    groupCondition=$(echo "$applicationGroupJson" | jq -r '.conditions[] | select(.reason=="Failed")')
    if [ -n "$groupCondition" ]; then
        testSuiteMessage "TEST_PASS" "ApplicationGroup status correct"
    else
        testSuiteMessage "TEST_FAIL" "ApplicationGroup status expected: (Failed)"
    fi
}

function runFailureScenarios {
    echo Running Failure Scenarios
    applyFailureOnExistingDeployment
    validateFailedApplicationGroup
    deployFailure
    validateFailedApplicationGroup
    summary
    echo DONE
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
runFailureScenarios
