#!/bin/bash
sudo docker exec -i smf-mysql mysql -uroot -proot smf < /tmp/smf-php/reset.sql
cd /tmp/smf-php && go run crawl.go
