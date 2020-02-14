# oci8 dialect

- [oci8 dialect](#oci8-dialect)
  - [Overview](#overview)
  - [Install oracle client (required by go module)](#install-oracle-client-required-by-go-module)
  - [Install the Module](#install-the-module)
    - [Basic Test](#basic-test)
  - [Running Oracle in Docker](#running-oracle-in-docker)


## Overview

This document covers everything you need to install/configure/run to get the oci8 dialect up and running.  BTW, oci8 using pkg-config to bind to the Oracle instantclient libraries, so it uses CGO.

You'll find some additional documentation on various challenges and design decisions for the dialect in [doc.go](./doc.go)

## Install oracle client (required by go module)

Download instaclients from: https://www.oracle.com/database/technologies/instant-client/macos-intel-x86-downloads.html

You need the basic, sqlplus and sdk packages.

Move the zip instaclient files to `~/Library/Caches/Homebrew`

Install the clients with brew

 ```
$ brew tap InstantClientTap/instantclient
$ brew install instantclient-basic
$ brew install instantclient-sqlplus
$ brew install instantclient-sdk
```

## Install the Module

```
go get github.com/mattn/go-oci8
```

Install pkg-config via brew

```
$ brew install pkg-config
```

Create `/usr/local/lib/pkgconfig/oci8.pc`
Contents:
```
prefixdir=/usr/local/lib
libdir=${prefixdir}
includedir=${prefixdir}/sdk/include

Name: OCI
Description: Oracle database driver
Version: 12.2
Libs: -L${libdir} -lclntsh
Cflags: -I${includedir}% 
``` 

Test pkg-config for oci8
```
$ pkg-config --cflags --libs oci8
-I/usr/local/lib/sdk/include -L/usr/local/lib -lclntsh
```

### Basic Test

Once you've got Oracle up and running and you've created the gorm user with appropriate privs, try to run:
```
# chdir to the oci8 dialect's directory:
$ cd dialect/oci8

# run a basic connection test
$ go test -v  -run Test_Connection
=== RUN   Test_Connection
--- PASS: Test_Connection (0.06s)
    connection_test.go:29: The date is: 2020-02-14T20:09:37Z
PASS
ok      github.com/jinzhu/gorm/dialects/oci8    1.609s
```

## Running Oracle in Docker

Download the Oracle Express Edition version 18c (xe) Linux rpm from oracle.com

```
git clone https://github.com/oracle/docker-images.git 
```

copy binary you downloaded
```
cp ~/Dowloads/<image-name> ./18.4.0
```
build image
```
./buildDockerImage.sh -x -v 18.4.0
```
start container
```
docker run -dit --name oracledb \
       -p 1521:1521 -p 5500:5500  -p 8080:8080 \
       -e ORACLE_PWD=oracle \
       -v $HOME/oracle/oradata:/opt/oracle/oradata \
       oracle/database:18.4.0-xe
```

Check logs and wait for oracle to be ready…
```
❯ docker logs -f oracledb
The Oracle base remains unchanged with value /opt/oracle
#########################
DATABASE IS READY TO USE!
#########################
 ```

 Connect via sqlplus as sysdba via container
```
❯ docker exec -it oracledb sqlplus sys/oracle@localhost:1521/XE as sysdba

SQL*Plus: Release 18.0.0.0.0 - Production on Mon Feb 10 16:33:44 2020
Version 18.4.0.0.0

Copyright (c) 1982, 2018, Oracle.  All rights reserved.


Connected to:
Oracle Database 18c Express Edition Release 18.0.0.0.0 - Production
Version 18.4.0.0.0

SQL> 
 ```

Connect via sqlplus as system user via container:
```
docker exec -it oracledb sqlplus system/oracle@localhost:1521/XE

SQL*Plus: Release 18.0.0.0.0 - Production on Mon Feb 10 17:08:04 2020
Version 18.4.0.0.0

Copyright (c) 1982, 2018, Oracle.  All rights reserved.

Last Successful login time: Mon Feb 10 2020 17:07:07 +00:00

Connected to:
Oracle Database 18c Express Edition Release 18.0.0.0.0 - Production
Version 18.4.0.0.0

SQL> 
 ```

create a gorm user 
```
docker exec -it oracledb sqlplus system/oracle@localhost:1521/XE

SQL> 
-- creating user in the default PDB
ALTER SESSION SET CONTAINER = XEPDB1;
create user gorm identified by gorm;
-- we need some privs
GRANT CONNECT, RESOURCE, DBA TO gorm;
GRANT CREATE SESSION TO gorm;
GRANT UNLIMITED TABLESPACE TO gorm;
```

