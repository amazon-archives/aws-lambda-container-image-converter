<?php

// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

function hello($data)
{
    $response = [
        'msg' => "Hello, {$data['name']}!",
        'data' => $data
    ];
    return $response;
}
