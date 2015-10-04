pg_dump --host=sql.microcosm.cc     --username=microcosm -s -f dump_prod.sql microcosm
pg_dump --host=sql.dev.microcosm.cc --username=microcosm -s -f dump_dev.sql  microcosm
java -jar apgdiff-2.4.jar --ignore-start-with dump_prod.sql dump_dev.sql > upgrade.sql
rm dump_prod.sql
rm dump_dev.sql
