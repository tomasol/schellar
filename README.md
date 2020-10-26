# schellar

[<img src="https://img.shields.io/docker/pulls/frinx/schellar"/>](https://hub.docker.com/r/frinx/schellar)
[<img src="https://img.shields.io/docker/automated/frinx/schellar"/>](https://hub.docker.com/r/frinx/schellar)

Schellar is a scheduler tool for invoking Conductor workflows from time to time

## Usage

* Checkout this repo

  * Needed just to use the sample Conductor workflow at "/example-conductor"

```
git clone github.com/frinxio/schellar

```

* Create/use docker-compose.yml file

```yml
version: '3.5'

services:

  schellar:
    image: frinx/schellar
    environment:
      - CONDUCTOR_API_URL=http://conductor-server:8080/api
      - MONGO_ADDRESS=mongo
      - MONGO_USERNAME=root
      - MONGO_PASSWORD=root
      - LOG_LEVEL=info
    ports:
      - 3000:3000
    logging:
      driver: "json-file"
      options:
        max-size: "20MB"
        max-file: "5"

  mongo:
    image: mongo:4.1.10
    environment:
      - MONGO_INITDB_ROOT_USERNAME=root
      - MONGO_INITDB_ROOT_PASSWORD=root
    ports:
      - 27017-27019:27017-27019

  mongo-express:
    image: mongo-express:0.49.0
    ports:
      - 8081:8081
    environment:
      - ME_CONFIG_MONGODB_ADMINUSERNAME=root
      - ME_CONFIG_MONGODB_ADMINPASSWORD=root

  conductor-server:
    build: example-conductor
    ports:
      - 8080:8080
    environment:
      - DYNOMITE_HOSTS=dynomite:8102:us-east-1c
      - ELASTICSEARCH_URL=elasticsearch:9300
      - LOADSAMPLE=false
      - PROVISIONING_UPDATE_EXISTING_TASKS=false

  dynomite:
    image: flaviostutz/dynomite:0.7.5
    ports:
      - 8102:8102

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:5.6.8
    environment:
      - "ES_JAVA_OPTS=-Xms512m -Xmx1000m"
      - transport.host=0.0.0.0
      - discovery.type=single-node
      - xpack.security.enabled=false
    ports:
      - 9200:9200
      - 9300:9300
    logging:
      driver: "json-file"
      options:
        max-size: "20MB"
        max-file: "5"

```

* Run "docker-compose up" and wait for the logs to calm down :)

* Create a new schedule to run the sample Conductor Workflow every minute

```
curl -X POST \
  http://localhost:3000/schedule \
  -H 'Content-Type: application/json' \
  -H 'cache-control: no-cache' \
  -d '{
	"name": "seconds-tests1",
	"enabled": true,
	"parallelRuns": false,
	"workflowName": "encode_and_deploy",
	"workflowVersion": "1",
	"cronString": "0 * * ? * *",
	"workflowContext": {
		"param1": "value1",
		"param2": "value2"
	},
	"fromDate": "2019-01-01T15:04:05Z",
	"toDate": "2029-07-01T15:04:05Z"
}
'
```

* Open http://localhost:8081 to access Mongo UI and access the collection "schedules" to view status

* Open http://localhost:5000 to access Conductor UI and view the workflow instances that were generated by Schellar

* As there is no Worker executing the Conductor tasks, Schellar will create only one workflow instance. In the example parallelRuns are disabled.

* If you Terminate the running Workflow in Conductor, Schellar will create another workflow instance when its timer triggers.

## API

  * **POST /schedule**
    * Creates a new schedule
    * JSON Body

```shell
  curl -X POST \
  http://localhost:3000/schedule \
  -H 'Content-Type: application/json' \
  -H 'cache-control: no-cache' \
  -d '{
	"name": "seconds-tests1",
	"enabled": true,
	"parallelRuns": false,
	"workflowName": "encode_and_deploy",
	"workflowVersion": "1",
	"cronString": "*/30 * * ? * *",
	"workflowContext": {
		"param1": "value1",
		"param2": "value2"
	},
	"fromDate": "2019-01-01T15:04:05Z",
	"toDate": "2019-07-01T15:04:05Z",
	"correlationId" "myid",
        "taskToDomain": {
		"*": "mydomain"
	}
      }'
```
* Where
  * **name** - schedule name (must be unique)
  * **enabled** - active or not
  * **cronString** - cron string specification of the timer used to trigger new Conductor workflows from time to time (see more at https://crontab.guru)
  * **fromDate** - start date to enable this schedule
  * **toDate** - end date to enable this schedule
  * **workflowName** - workflow name that will be instantiated in Conductor
  * **workflowVersion** - workflow version in Conductor
  * **workflowContext** - key/value in json style used as input for new workflow instances.
    * When a workflow instance is COMPLETED, its output values will be added to the current schedule workflow context under `lastExecution` attribute so that these new values will be used on the next workflow instantiation calls as "input".
    * This may be useful in cases where your workers want to return data that will be used on following workflow calls. For example, workflow instance 1 will process from date 2019-01-01 to 2019-01-15 and its output will be lastDate=2019-01-15; than instance2 from 2019-01-16 to 2019-02-11 and returns lastDate=2019-02-11 and so on.
  * **parallelRuns** - if true, every trigger from timer (according to cron string) will generate a new workflow instance in Conductor. if false, no new workflows will be generated if there are other workflow instances in state RUNNING, so that only one RUNNING instance will be present at a time
  * **correlationId** - passed to Conductor when starting a workflow, see https://netflix.github.io/conductor/gettingstarted/startworkflow/
  * **taskToDomain** - passed to Conductor when starting a workflow, see https://netflix.github.io/conductor/configuration/taskdomains/

  * **GET /schedule**
    * Returns a list of schedules

  * **PUT /schedule/{schedule-name}**
    * Updates existing schedules (updating active timers accordingly)
    * JSON Body with contents that would be updated

```shell
curl -X PUT \
  http://localhost:3000/schedule/seconds-tests1 \
  -H 'Content-Type: application/json' \
  -H 'cache-control: no-cache' \
  -d '{
	"enabled": true,
	"cronString": "*/45 * * ? * *"
      }'
```

## ENV configurations
Schellar is configured using [GoDotEnv](https://github.com/joho/godotenv).

See [.env-SAMPLE](schellar/.env-SAMPLE) file for sample configuration.

## DB migrations

Schellar uses [tern](https://github.com/jackc/tern) for DB migrations.
They run automatically when schellar starts. It is possible to use a
standalone cli tool to manage migratinos as well:
```sh
go get -u github.com/jackc/tern
# execute migrations
cd schellar/migrations
tern migrate --version-table schellar_schema_version
```

## Testing
To run tests, first go to `schellar` directory.
To run unit tests:
```sh
go test -v -short ./...
```
To run integration tests:
```sh
docker-compose -f docker-compose.test.yml up -d
export POSTGRES_DATABASE_URL="host=127.0.0.1 port=6432 user=postgres password=postgres database=schellar_test"
export POSTGRES_MIGRATIONS_DIR="$(pwd)/migrations"
go test -run Integration ./...
```
