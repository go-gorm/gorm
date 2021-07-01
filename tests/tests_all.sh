#!/bin/bash -e

dialects=("sqlite" "mysql" "postgres" "sqlserver")

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

if [ -d tests ]
then
  cd tests
  go get -u ./...
  go mod download
  go mod tidy
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
