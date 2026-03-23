module github.com/nixys/nxs-data-anonymizer

go 1.26.1

require (
	github.com/Masterminds/sprig/v3 v3.3.0
	github.com/docker/go-units v0.5.0
	github.com/go-sql-driver/mysql v1.9.3
	github.com/jaswdr/faker/v2 v2.9.1
	github.com/nixys/nxs-go-appctx/v3 v3.0.0
	github.com/nixys/nxs-go-conf v1.3.0
	github.com/nixys/nxs-go-fsm v1.0.0
	github.com/pborman/getopt/v2 v2.1.0
	github.com/sirupsen/logrus v1.9.4
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/coregx/ahocorasick v0.2.1 // indirect
	github.com/cskr/pubsub v1.0.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/nixys/nxs-go-fsm => github.com/screechingbagel/nxs-go-fsm v0.0.0-20260323225450-858ea73512a4
