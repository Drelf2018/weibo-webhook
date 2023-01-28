package main

func checkErr(err error) bool {
	if err != nil {
		panic(err)
	} else {
		return true
	}
}
