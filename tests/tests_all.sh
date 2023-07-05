#!/bin/bash -e

dialects=("sqlite" "mysql" "postgres" "sqlserver" "tidb")

if [[ $(pwd) == *"gorm/tests"* ]]; then
	cd ..
fi

if [ -d tests ]; then
	cd tests
	go get -u -t ./...
	go mod download
	go mod tidy
	cd ..
fi

# SqlServer for Mac M1
if [[ -z $GITHUB_ACTION ]]; then
	if [ -d tests ]; then
		cd tests
		if [[ $(uname -a) == *" arm64" ]]; then
			MSSQL_IMAGE=mcr.microsoft.com/azure-sql-edge docker-compose start || true
			go install github.com/microsoft/go-sqlcmd/cmd/sqlcmd@latest || true
			SQLCMDPASSWORD=LoremIpsum86 sqlcmd -U sa -S localhost:9930 -Q "IF DB_ID('gorm') IS NULL CREATE DATABASE gorm" >/dev/null || true
			SQLCMDPASSWORD=LoremIpsum86 sqlcmd -U sa -S localhost:9930 -Q "IF SUSER_ID (N'gorm') IS NULL CREATE LOGIN gorm WITH PASSWORD = 'LoremIpsum86';" >/dev/null || true
			SQLCMDPASSWORD=LoremIpsum86 sqlcmd -U sa -S localhost:9930 -Q "IF USER_ID (N'gorm') IS NULL CREATE USER gorm FROM LOGIN gorm; ALTER SERVER ROLE sysadmin ADD MEMBER [gorm];" >/dev/null || true
		else
			docker-compose start
		fi
		cd ..
	fi
fi

for dialect in "${dialects[@]}"; do
	if [ "$GORM_DIALECT" = "" ] || [ "$GORM_DIALECT" = "${dialect}" ]; then
		echo "testing ${dialect}..."

		cmd="GORM_DIALECT=${dialect} go test"
		tags=("")

		if [ "$GORM_DIALECT" = "sqlite" ]; then
			# Test SQLite pure-go driver
			tags+=("pure")
		fi

		for tag in "${tags[@]}"; do
			echo "testing ${dialect} with tag '${tag}'"

			if [ -n "$tag" ]; then
				cmd="$cmd -tags ${tag}"
			fi

			cmd="$cmd -race -count=1 ./..."


			if [ "$GORM_VERBOSE" = "" ]; then
				eval $cmd
				if [ -d tests ]; then
					cd tests
					eval $cmd
					cd ..
				fi
			else
				eval $cmd
				if [ -d tests ]; then
					cd tests
					eval $cmd
					cd ..
				fi
			fi
		done
	fi
done
