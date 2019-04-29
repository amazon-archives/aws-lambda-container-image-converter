<?php

// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

function goodbye($data)
{
    $response = [
        'msg' => "Goodbye, {$data['name']}!",
        'data' => $data
    ];
    return $response;
}
