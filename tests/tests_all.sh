dialects=("postgres" "mysql" "mssql" "sqlite")

if [[ $(pwd) == *"gorm/tests"* ]]; then
  cd ..
fi

for dialect in "${dialects[@]}" ; do
  if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]
  then
    if [ "$GORM_VERBOSE" = "" ]
    then
      cd dialects/${dialect}
      DEBUG=false GORM_DIALECT=${dialect} go test -race ./...
      cd ../..

      DEBUG=false GORM_DIALECT=${dialect} go test -race ./...
    else
      cd dialects/${dialect}
      DEBUG=false GORM_DIALECT=${dialect} go test -race ./...
      cd ../..

      DEBUG=false GORM_DIALECT=${dialect} go test -race -v ./...
    fi
  fi
done
