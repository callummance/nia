#!/bin/bash
HOST=$1
ssh -o StrictHostKeyChecking=no -p 2222 -L 8080:rethinkdb:8080 -L 28015:rethinkdb:28015 nia@$HOST