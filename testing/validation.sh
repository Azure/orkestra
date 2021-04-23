#!/bin/bash
ORKESTRA_RESOURCE_COUNT=5
g_successCount=0
g_failureCount=0

function outputMessage {
    if [ "$1" == "SUCCESS" ]; then
        echo SUCCESS: $2 
        ((g_successCount++))
    else
        echo FAIL: $2
        ((g_failureCount++))
    fi
}

function summary {
    echo Success Cases: $g_successCount
    echo Failure Cases: $g_failureCount
}

function validateOrkestraDeployment {
    resources=$(kubectl get pods --namespace orkestra 2>> log.txt| grep -i -c running)
    if [ $resources -ne $ORKESTRA_RESOURCE_COUNT ]; then
        outputMessage "FAIL" "No running orkestra resources. Currently $resources running resources. Expected $ORKESTRA_RESOURCE_COUNT"
    else
        outputMessage "SUCCESS" "orkestra resources are running"
    fi

    orkestraStatus=$(helm status orkestra -n orkestra 2>> log.txt | grep -c deployed)
    if [ $orkestraStatus -eq 1 ]; then
        outputMessage "SUCCESS" "orkestra deployed successfully"
    else
        outputMessage "FAIL" "orkestra not deployed"
    fi
}

function validateBookInfoDeployment {
    ambassadorStatus=$(helm status ambassador -n ambassador 2>> log.txt | grep -c deployed)
    if [ $ambassadorStatus -eq 1 ]; then
        outputMessage "SUCCESS" "ambassador deployed successfully"
    else
        outputMessage "FAIL" "ambassador not deployed"
    fi

    bookinfoReleaseNames=("details" "productpage" "ratings" "reviews" "bookinfo")

    for var in "${bookinfoReleaseNames[@]}"
    do  
        deployedStatus=$(helm status $var -n bookinfo 2>> log.txt | grep -c deployed)
        if [ $deployedStatus -eq 1 ]; then
            outputMessage "SUCCESS" "$var deployed successfully"
        else
            outputMessage "FAIL" "$var not deployed"
        fi
    done
}

function validateArgoWorkflow {
    bookinfoStatus=$(curl -s --request GET --url http://localhost:2746/api/v1/workflows/orkestra/bookinfo | grep -c "not found")
    if [ "$bookinfoStatus" -eq 1 ]; then
        outputMessage "FAIL" "No argo workflow found for bookinfo"
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
                outputMessage "SUCCESS" "argo node: $node has succeeded"
            else
                outputMessage "FAIL" "$node status: $status, Expected Succeeded"
            fi
        done
    fi
}

function validateApplicationGroup {
    echo "Validating bookinfo applicationgroup"
    bookInfoGroupJson=$(kubectl get applicationgroup bookinfo -o json | jq '.spec.applications')
    
    temp=$(echo "$bookInfoGroupJson" | jq -r '.[0].name')
    if [ "$temp" == "ambassador" ]; then
        outputMessage "SUCCESS" "ambassador application found"
    else
        outputMessage "FAIL" "expected ambassador got $temp"
    fi

    temp=$(echo "$bookInfoGroupJson" | jq -r '.[1].name')
    if [ "$temp" == "bookinfo" ]; then
        outputMessage "SUCCESS" "bookinfo application found"
    else
        outputMessage "FAIL" "expected bookinfo got $temp"
    fi

    subcharts=("details" "productpage" "ratings" "reviews")

    for chart in "${subcharts[@]}"
    do
        temp=$(echo "$bookInfoGroupJson" | jq --arg c "$chart" -c '.[1].spec.subcharts[] | select(.name==$c)')
        if [ "$chart" == "productpage" ]; then
            dependency=$(echo "$temp" | jq -r '.dependencies | index( "reviews" )' )
            if [ "$dependency" == "null" ] || [ -z "$dependency" ]; then
                outputMessage "FAIL" "productpage dependency: reviews not found"
            else
                outputMessage "SUCCESS" "productpage dependency: reviews found"
            fi
        elif [ "$chart" == "reviews" ]; then 
            dependency=$(echo "$temp" | jq -r '.dependencies | index( "details" )' )
            if [ "$dependency" == "null" ] || [ -z "$dependency" ]; then
                outputMessage "FAIL" "reviews dependency: details not found"
            else
                outputMessage "SUCCESS" "reviews dependency: details found"
            fi

            dependency=$(echo "$temp" | jq -r '.dependencies | index( "ratings" )' )
            if [ "$dependency" == "null" ] || [ -z "$dependency" ]; then
                outputMessage "FAIL" "reviews dependency: ratings not found"
            else
                outputMessage "SUCCESS" "reviews dependency: ratings found"
            fi
        fi
        
        if [ -z $temp ]; then
            outputMessage "FAIL" "Did not find $chart in subcharts"
        else
            outputMessage "SUCCESS" "$chart in subcharts"
        fi
    done
}

function runValidation {
    echo Running Validation
    validateOrkestraDeployment
    validateBookInfoDeployment
    validateArgoWorkflow
    validateApplicationGroup
    summary
}

runValidation
