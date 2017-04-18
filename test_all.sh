dialects=("postgres" "mysql" "mssql" "sqlite")

for dialect in "${dialects[@]}" ; do
    GORM_DIALECT=${dialect} go test
done
