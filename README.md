# qms

qms is the CyVerse Quota Management System. Its purpose is to keep track of resource usage limits and totals for
CyVerse users.

## Concepts

### Plans

Plans define sets of default resource usage limits that can be assigned to CyVerse users. Every CyVerse user is
initially assigned a basic plan, and can choose to purchase a plan that provides more resources if the basic plan
is not adequate. This software does not manage purchases; it only keeps track of the plan that is currently active
for each user.

### Quotas

Quotas are resource usage limits that can be assigned to users for each resource type that is tracked in the system. In
most cases, the current quota that is applied to a user comes directly from the plan that is currently active for the
user. Quotas can be customized if necessary, but customizing quotas should be a rare occurrence.

### Current Usage

The qms tracks the current resource usage totals for each CyVerse user. These usage totals are calculated by other
CyVerse microservices and reported to the qms.

### Updates

Updates to both quotas and resource usage totals are recorded in the qms database for auditing purposes.

## Configuration Settings

The qms uses environment variables for its configuration settings. The following configuration settings are supported.

### QMS_DATABASE_URI (Required)

This environment variable contains the database connection URI. The qms uses a PostgreSQL database to keep track of
its data. Any version of PostgreSQL that supports the required extensions should work, but the qms has been tested
against versions 12, 13, and 14. The following extensions are required:

1. uuid-ossp
2. btree_gist
3. moddatetime
4. insert_username

The URI format is as follows:

```
postgresql://{db-username}:{db-password}@{db-host}:{db-port}/{db_name}?{options}
```

If SSL is not set up for the PostgreSQL instance that is being used then the `sslmode` option should explicitly be
disabled in the URI. Also, the password is optional if you have a [.pgpass file][1] containing the appropriate
connection settings. In most cases, the URI will look something like this:

```
postgresql://dbuser@dbhost.example.org:5432/dbname?sslmode=disable
```

### QMS_DATABASE_MIGRATE (Optional, Default: `false`)

If this environment variable is defined and set to `true` then qms will automatically run database schema migrations
on startup.

### QMS_DATABASE_REINIT (Optional, Default: `false`)

If this environment variable is defined and set to `true` then the qms will reinitialize the database upon startup.
This feature is intended to be used only during development testing, when the schema migrations are being actively
updated. Note: this parameter is only applicable if `QMS_DATABASE_MIGRATE` is also enabled.

## Database Schema Migraions

The qms runs its schema migrations upon startup. For this to succeed, two prerequisites must be satisfied. The first
requirement is that the database must exist and must have the required extensions listed above installed. Having the
extensions installed in advance allows qms to use a regular PostgreSQL account instead of requiring an administrative
account.

The second requirement is that the schema migrations must exist in a subdirectory of the current working directory
called `migrations`. At some point in the future, this requirement will likely be replaced by a configuration setting
that specifies a URI that can be used to locate the schema migrations.

[1]: https://www.postgresql.org/docs/current/libpq-pgpass.html
