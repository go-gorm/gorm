dialects=("postgres" "mysql" "sqlite" "mssql")

for dialect in "${dialects[@]}" ; do
    GORM_DIALECT=${dialect} go test
done
