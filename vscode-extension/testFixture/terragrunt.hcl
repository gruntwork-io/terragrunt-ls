locals {
  greeting = "hello"
}

inputs = {
  greeting = local.greeting
  unit_dir = path.unit
}
