# Sage [![Build Status](https://travis-ci.com/JohnStarich/sage.svg?branch=master)](https://travis-ci.com/JohnStarich/sage) [![Coverage Status](https://coveralls.io/repos/github/JohnStarich/sage/badge.svg?branch=master)](https://coveralls.io/github/JohnStarich/sage?branch=master)

Be your own accountant, without the stress.

Examine your finances with ease.
Automatically download transactions from your banks and credit cards, then run the numbers.

Get the latest release [here](#install), then [let us know what you think][feedback]!

[feedback]: https://github.com/JohnStarich/sage/issues/new

![Activity page demo](.github/media/activity.png)
<p align="center"><em>See your latest balances, expenses, and transactions.</em></p>

## Features

* [x] Securely & automatically download data directly from your bank or credit card company
* [x] View and edit transactions, balances, and budgets
* [x] Uses [double-entry bookkeeping][] to keep things in check
* [x] Web Connect <sup>(beta)</sup> and Direct Connect support
* [x] Can deploy as a single binary or as a Docker container
* [x] Automatic version control

![Budgets page demo](.github/media/budgets.png)
<p align="center"><em>Manage monthly budgets to keep track of your expenses.</em></p>

[double-entry bookkeeping]: https://en.wikipedia.org/wiki/Double-entry_bookkeeping_system

## Install

Choose **_one_** of the following options:

* Download the app for [Windows][], [Mac][], or [Linux][]
* Run the container image from [Docker Hub](https://hub.docker.com/r/johnstarich/sage):
```bash
DATA_DIR=$HOME/sage
mkdir "$DATA_DIR"
docker run \
    --detach \
    --name sage \
    --publish 127.0.0.1:8080:8080 \
    --volume "$DATA_DIR":/data \
    johnstarich/sage
# Visit http://localhost:8080 in your browser
```
* Download and install the latest Sage server release from the [releases page](https://github.com/JohnStarich/sage/releases/latest) or this script:
```bash
curl -fsSL -H 'Accept: application/vnd.github.v3+json' https://api.github.com/repos/JohnStarich/sage/releases/latest | grep browser_download_url | cut -d '"' -f 4 | grep -i "$(uname -s)-$(uname -m)" | xargs curl -fSL -o sage
chmod +x sage
./sage -help  # Optionally move sage into your PATH
```
* OR download the source and build it: `go get github.com/johnstarich/sage`

[Windows]: https://github.com/JohnStarich/sage/releases/latest/download/Sage-for-Windows.exe
[Mac]: https://github.com/JohnStarich/sage/releases/latest/download/Sage-for-Mac.zip
[Linux]: https://github.com/JohnStarich/sage/releases/latest/download/Sage-for-Linux.deb


## Usage

For available options, run `sage -help`

## Future work

* Over-budget notifications
* Forecasts on current transactions to identify trends
* Smarter categorization by training on current ledger

## Data storage

Sage uses a ledger ([plain text accounting][]) file, some simple JSON-encoded files, and an [`hledger` rules][hledger rules] file.
**You won't need to know about these files to use Sage.** However, if you're a power-user, then these formats may come in handy.

[plain text accounting]: https://plaintextaccounting.org
[hledger rules]: https://hledger.org/csv.html#csv-rules

The ledger will store all of your transactions in plain text so you can easily read it with any text editor. It also supports [several other tools][ledger tools] that can generate reports based on your ledger.

**Warning:** Some banks, like [Bank of America][], may charge a fee for downloading transactions. While this is uncommon, we are not responsible for these charges. Do your homework if you want to be certain these charges won't apply to you.

[Bank of America]: https://wiki.gnucash.org/wiki/OFX_Direct_Connect_Bank_Settings#BofA.2C_CA

The rules file is a format designed by the [hledger][] project for importing CSVs. This file will help Sage automatically categorize incoming transactions into the appropriate accounts for your ledger. After a transaction has been imported, it is assigned an account (category) from this file. To follow convention, only include rules to change the `account2` field or a `comment`. While changing `account1` is supported, it will likely cause problems with Sage since account1 is assumed to be the source institution of the transaction.
Currently, the web UI only supports `account2`.

[hledger]: https://github.com/simonmichael/hledger
[ledger tools]: https://plaintextaccounting.org/#plain-text-accounting-tools

## Awesome libraries 👏

Sage relies on [`aclindsa/ofxgo`](https://github.com/aclindsa/ofxgo) for it's excellent Go implementation of the OFX spec.
