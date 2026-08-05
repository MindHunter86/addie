module github.com/MindHunter86/addie

go 1.22.12

exclude (
	golang.org/x/exp v0.0.0-20251009144603-d2f985daa21b
	golang.org/x/exp v0.0.0-20251125195548-87e1e737ad39
	golang.org/x/exp v0.0.0-20260727155853-b88d891fe743
)

exclude (
	github.com/hashicorp/serf v0.10.3
	github.com/hashicorp/serf v0.10.4
)

exclude (
	github.com/mattn/go-runewidth v0.0.26
	github.com/mattn/go-runewidth v0.0.27
)

exclude go.etcd.io/bbolt v1.3.12

replace (
	github.com/armon/go-metrics v0.4.1 => github.com/hashicorp/go-metrics v0.4.1
	github.com/armon/go-metrics v0.4.2 => github.com/hashicorp/go-metrics v0.4.2
)

exclude (
	github.com/fsnotify/fsnotify v1.10.0
	github.com/fsnotify/fsnotify v1.10.1
)

require (
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-kit/kit v0.9.0
	github.com/go-ozzo/ozzo-validation/v4 v4.4.1
	github.com/go-playground/validator/v10 v10.27.0
	github.com/gofiber/fiber/v2 v2.52.14
	github.com/jedib0t/go-pretty/v6 v6.8.3
	github.com/k0kubun/pp v3.0.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/rs/zerolog v1.33.0
	github.com/spaolacci/murmur3 v1.1.0
	github.com/urfave/cli/v2 v2.27.7
	github.com/valyala/bytebufferpool v1.0.0
	github.com/valyala/fasthttp v1.51.0
	github.com/valyala/tcplisten v1.0.0
	go.uber.org/atomic v1.11.0
	golang.org/x/net v0.35.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/andybalholm/brotli v1.1.1 // indirect
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/clipperhouse/uax29/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-logfmt/logfmt v0.5.1 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/k0kubun/colorstring v0.0.0-20150214042306-9440f1994b88 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.15 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mattn/go-runewidth v0.0.25 // indirect
	github.com/philhofer/fwd v1.1.3-0.20240916144458-20a13a1f6b7c // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tinylib/msgp v1.2.5 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)

exclude github.com/leodido/go-urn v1.5.0

exclude (
	github.com/go-playground/validator/v10 v10.28.0
	github.com/go-playground/validator/v10 v10.29.0
	github.com/go-playground/validator/v10 v10.30.0
	github.com/go-playground/validator/v10 v10.30.1
	github.com/go-playground/validator/v10 v10.30.2
	github.com/go-playground/validator/v10 v10.30.3
)
