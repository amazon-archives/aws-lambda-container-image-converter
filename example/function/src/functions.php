<?php

// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

use LambdaPHPRuntime\Context;

/**
 * @param $event
 * @return string
 */

function hello($event)
{
    if (isset($event['name'])) {
        return "Hello, {$event['name']}!";
    } else {
        return "Hello, World!";
    }
}

/**
 * @param $event
 * @return string
 */

function goodbye($event)
{
    if (isset($event['name'])) {
        return "Goodbye, {$event['name']}!";
    } else {
        return "Goodbye";
    }
}

/**
 * @param $event
 * @param Context $context
 */
function diagnostics($event, Context $context) {
    print_r($event);

    echo "Execution Time Remaining:" . $context->getRemainingTimeInMillis() . "\n";

    echo "Request ID:" . $context->getAwsRequestId() . "\n";
    echo "Invoked Function ARN:" . $context->getInvokedFunctionArn() . "\n";
    echo "Log Group Name:" . $context->getLogGroupName() . "\n";
    echo "Log Stream Name:" . $context->getLogStreamName() . "\n";
    echo "Function Name:" . $context->getFunctionName() . "\n";
    echo "Function Version:" . $context->getFunctionVersion() . "\n";
    echo "Memory Limit (MB):" . $context->getMemoryLimitInMB() . "\n";
    echo "Client Context:" . $context->getClientContext() . "\n";
    echo "Identity (Cognito):" . $context->getIdentity() . "\n";
    echo "Execution Time Remaining:" . $context->getRemainingTimeInMillis() . "\n";
}
?>