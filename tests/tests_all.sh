#!/bin/bash -e

dialects=("sqlite" "mysql" "postgres" "sqlserver" "oracle")

ORACLE_INSTANT_CLIENT_URL="https://download.oracle.com/otn_software/linux/instantclient/19600/instantclient-basic-linux.x64-19.6.0.0.0dbru.zip"
ORACLE_INSTANT_CLIENT_FILE="instant_client.zip"

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

if [ -d tests ]
then
  cd tests
  cp go.mod go.mod.bak
  sed '/$[[:space:]]*gorm.io\/driver/d' go.mod.bak > go.mod
  cd ..
fi

for dialect in "${dialects[@]}" ; do
  if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]
  then
    if [[ "$dialect" =~ "oracle" ]]
    then
      if [[ ! -d $(pwd)/instantclient_19_6 ]]
      then
        if [[ ! -f "$ORACLE_INSTANT_CLIENT_FILE" ]]
        then
          echo "downloading oracle instant client..."
          curl "$ORACLE_INSTANT_CLIENT_URL" -o "$ORACLE_INSTANT_CLIENT_FILE"
        fi
        echo "unzipping oracle instant client..."
        unzip -o "$ORACLE_INSTANT_CLIENT_FILE"
      fi
      export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:$(pwd)/instantclient_19_6
      echo "exported instant client libraries to LD_LIBRARY_PATH, now it should not complain about missing oracle libraries"
    fi
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

if [ -d tests ]
then
  cd tests
  mv go.mod.bak go.mod
fi
