dialects=("postgres" "mysql" "sqlite", "tidb")

for dialect in "${dialects[@]}" ; do
    GORM_DIALECT=${dialect} go test
done
