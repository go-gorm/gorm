dialects=("postgres" "mysql" "mssql" "sqlite", "oci8")

for dialect in "${dialects[@]}" ; do
    DEBUG=false GORM_DIALECT=${dialect} go test
done
