dialects=("postgres" "mysql" "sqlite" "cockroach")

for dialect in "${dialects[@]}" ; do
    GORM_DIALECT=${dialect} go test
done
