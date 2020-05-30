dialects=("sqlite" "mysql" "postgres" "mssql")

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

for dialect in "${dialects[@]}" ; do
  if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]
  then
    echo "testing ${dialect}..."

    if [ "$GORM_VERBOSE" = "" ]
    then
      DEBUG=false GORM_DIALECT=${dialect} go test -race -count=1 ./...
    else
      DEBUG=false GORM_DIALECT=${dialect} go test -race -count=1 -v ./...
    fi
  fi
done
