# AusGridBot

This is a bot that reads electricity pricing information from AEMO's API
and posts interesting things to Mastodon. Currently it only posts updates
to the pricing in Queensland, but it could be expanded to other states if
there's interest.

## Hosting

This repo is set up to be run on a fly.io instance to make self-hosting easy. The basic free account is good enough for this purpose.

The bring-up process is pretty trivial:

    1. Install git and flyctl
        `sudo apt install git`
        `curl -L https://fly.io/install.sh | sh`
    2. Clone this repo
        `git clone git@github.com:tjhowse/bomgifbot.git`
    3. `flyctl auth signup`
    4. Set the config in the `env` section of `fly.toml`
    5. `flyctl launch`
    6. Set all the secrets listed in secrets.toml.template
        `flyctl secrets set MASTODON_CLIENT_ID=<your client id> MASTODON_CLIENT_SECRET=<your client secret>`, etc
    7. `flyctl deploy`

To pause the app, run `flyctl scale count 0`. To resume, run `flyctl deploy`. There might be steps missing here. I wrote this post-hoc.

## Configuration

| Setting | Description | Secret | Example | Default |
| --- | --- | --- | --- | --- |
| `MASTODON_SERVER` | The URL of the mastodon server to post to | No | `https://botsin.space` | N/A |
| `MASTODON_CLIENT_ID` | The client ID of the mastodon app to use | Yes | `1234567890` | N/A |
| `MASTODON_CLIENT_SECRET` | The client secret of the mastodon app to use | Yes | `1234567890` | N/A |
| `MASTODON_USER_EMAIL` | The email address of the mastodon account | Yes | `woo@you.com` | N/A |
| `MASTODON_USER_PASSWORD` | The user password of the mastodon account | Yes | `1234567890` | N/A |
| `GRID_BOT_CREDENTIALS` | A JSON-formatted list of per-region credentials. The above credentials are used if this is blank. | Yes | See below | "" |
| `AEMO_CHECK_INTERVAL` | The number of seconds between checking the AEMO API for new forecast information | No | `1200` | `1200` |
| `TEST_MODE` | If true, do not toot anything to mastodon, just log messages | No | `true` | `false` |


### Example GridBot credentials json

```json
[
    {
        "RegionID": "QLD1",
        "MastodonClientID": "clientid",
        "MastodonClientSecret": "clientsecret",
        "MastodonUserEmail": "useremail",
        "MastodonUserPassword": "userpassword"
    },
    {
        "RegionID": "NSW1",
        "MastodonClientID": "clientid",
        "MastodonClientSecret": "clientsecret",
        "MastodonUserEmail": "useremail",
        "MastodonUserPassword": "userpassword"
    }
]
```