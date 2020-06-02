dialects=("sqlite" "mysql" "postgres" "sqlserver")

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

for dialect in "${dialects[@]}" ; do
  if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]
  then
    echo "testing ${dialect}..."

    race=""
    if [ "$GORM_DIALECT" = "sqlserver" ]
    then
      race="-race"
    fi

    if [ "$GORM_VERBOSE" = "" ]
    then
      DEBUG=false GORM_DIALECT=${dialect} go test $race -count=1 ./...
      cd tests
      DEBUG=false GORM_DIALECT=${dialect} go test $race -count=1 ./...
    else
      DEBUG=false GORM_DIALECT=${dialect} go test $race -count=1 -v ./...
      cd tests
      DEBUG=false GORM_DIALECT=${dialect} go test $race -count=1 -v ./...
    fi
    cd ..
  fi
done
