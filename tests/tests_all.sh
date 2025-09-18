#!/bin/bash -e

dialects=("sqlite" "mysql" "postgres" "gaussdb" "sqlserver" "tidb")

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

if [ -d tests ]
then
  cd tests
  go get -u -t ./...
  go mod download
  go mod tidy
  cd ..
fi

# SqlServer for Mac M1
if [[ -z $GITHUB_ACTION && -d tests ]]; then
  cd tests
  if [[ $(uname -a) == *" arm64" ]]; then
    MSSQL_IMAGE=mcr.microsoft.com/azure-sql-edge docker compose up -d --wait
    go install github.com/microsoft/go-sqlcmd/cmd/sqlcmd@latest || true
    for query in \
      "IF DB_ID('gorm') IS NULL CREATE DATABASE gorm" \
      "IF SUSER_ID (N'gorm') IS NULL CREATE LOGIN gorm WITH PASSWORD = 'LoremIpsum86';" \
      "IF USER_ID (N'gorm') IS NULL CREATE USER gorm FROM LOGIN gorm; ALTER SERVER ROLE sysadmin ADD MEMBER [gorm];"
    do
      SQLCMDPASSWORD=LoremIpsum86 sqlcmd -U sa -S localhost:9930 -Q "$query" > /dev/null || true
    done
  else
    MSSQL_IMAGE=mcr.microsoft.com/mssql/server docker compose up -d --wait
  fi
  cd ..
fi


for dialect in "${dialects[@]}" ; do
  if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]
  then
    echo "testing ${dialect}..."

    if [ "$GORM_VERBOSE" = "" ]
    then
      GORM_DIALECT=${dialect} go test -race -count=1 ./...
      if [ -d tests ]
      then
        cd tests
        GORM_DIALECT=${dialect} go test -race -count=1 ./...
        cd ..
      fi
    else
      GORM_DIALECT=${dialect} go test -race -count=1 -v ./...
      if [ -d tests ]
      then
        cd tests
        GORM_DIALECT=${dialect} go test -race -count=1 -v ./...
        cd ..
      fi
    fi
  fi
done
