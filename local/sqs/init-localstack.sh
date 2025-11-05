#!/bin/bash

awslocal sqs create-queue --queue-name test-queue
awslocal sqs create-queue --queue-name product-notifications
