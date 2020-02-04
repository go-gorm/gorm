package clause

type OnConflict struct {
	ON     string  // duplicate key
	Values *Values // update c=c+1
}
