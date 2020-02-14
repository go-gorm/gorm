package oci8

/*

Understanding Oracle

It's a bit different than the other RDBMS databases and I'll just try to
hightlight a few of the important ones the dialect has to deal with:

1. Oracle upper cases all non-quoted identifiers. That means the dialect
has to decide what to do:
	1. quote all identifiers which would require developers to quote every
		identifer they passed in a string to gorm.
	2. only quote identifers that conflict with reserved words and leave all
		other identifiers unquoted, which means Oracle will automatically
		upper case them.  This would allow developers to pass unquoted
		identifiers in strings they passed to gorm and make the experience
		align better with the other dialects.
We chose option #2.

This design decision has the following side affects:
	a. you must be case insensitive when matching column names, like in
		the Scope.scan function
	b. Devs will have to escape reserved words when they reference them
		in things like: First(&CreditCard{}, `"number" = ?`)


2. Oracle handles last inserted id a bit differently, and requires a sql.Out
parameter to return the value.  Since Oracle parameters are positional, you
need to know how many other bind variables there are before adding the returning
clause.  (see createCallback() )

3. Oracle doesn't let you specify "AS <tablename> " when selecting a count
	from a dynamic table, so you just omit it. (see Scope.count() )

4. Oracle handles foreign keys a bit differently:
	A. REFERENCES is implicit
	B. ON UPDATE is not supported
	(see scope.addForeignKey() )

5. Searching a blob requires using a function from the dbms_lob package like
	instr() and specifying the offset and number of matches.
	(see oci8.SearchBlob() )

6 Trailing semicolons are not allowed at the end of Oracle sql statements
	(so they were removed in the unit tests)

*/
