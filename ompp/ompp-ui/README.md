# OpenM++ user interface (ompp-ui)

This repository is a part of [OpenM++](http://www.openmpp.org/) open source microsimulation platform.
It contains openM++ user interface.

## Project setup
```
git clone https://github.com/openmpp/UI.git ompp-ui
cd ompp-ui
npm install
```
(optional) audit packages:
```
npm audit --omit=dev
```

### Start the app in development mode (hot-code reloading, error reporting, etc.)
```
npm run dev
```
_2026-03-05: Vite `dev` is unstable, please watch for load errors, you may need to visit the same route multiple times before it is actually loads all assets._

### Build the app for production
```
npm run build
```

### Lint the files
```
npm run lint
```

*You can also use `bun` instead of `npm` for faster development.*

**License:** MIT.

Please visit our [wiki](https://github.com/openmpp/openmpp.github.io/wiki) for more information or e-mail to: _openmpp dot org at gmail dot com_.
