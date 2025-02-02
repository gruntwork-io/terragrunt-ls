locals {
	foo = "bar"
	map = {
		foo = "bar"
	}
	encoded = jsonencode(local.map)
}

inputs = {
	foo = local.foo
	map = local.map
	encoded = local.encoded
}
