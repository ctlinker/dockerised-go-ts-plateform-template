env {
  name = "local"
  url = "" # Replace with your DB URL
  src = "./sequel/schema.sql" # Path to your schema file
  dst = "./migrations" # Where migrations will be generated
}