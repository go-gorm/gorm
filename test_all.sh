dialects=("postgres" "mysql" "sqlite" "ql")

for dialect in "${dialects[@]}" ; do
    GORM_DIALECT=${dialect} go test
done
