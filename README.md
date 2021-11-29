# Postgres

1. docker pull postgres 
2. docker run -d --name my_postgres -v my_dbdata:/var/lib/postgresql/data -p 54320:5432 -e POSTGRES_PASSWORD=my_password postgres
3. docker exec -it my_postgres bash
4. psql -h localhost -U postgres
5. \l -->list databases
6. create database qmsdb --> to create database
7. \password --> to Set passowrd
8. Optional: docker run -d -e PGADMIN_DEFAULT_EMAIL=admin@example.com -e PGADMIN_DEFAULT_PASSWORD=admin -p 8000:80 dpage/pgadmin4