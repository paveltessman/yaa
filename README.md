# Yet Another Agent

How difficult can it be to build an LLM-powered app in the Go ecosystem? Well, let's find out!

`yaa` is a personal agent. You talk to it in plain language, and it uses tools to do the work - as simple as that.

The first UI will be a Telegram bot. The tools I'll add first are reminders, a reading list, and a place for the posts I forward to it. More will follow.

## How it works

All the work is done in pipelines. Pipelines can be viewed as domains. Each pipeline encapsulates and isolates some part of the work. For example, an agent pipeline knows about the agent, tool and other llm stuff, and knows nothing about telegram, transport and other layers. `telegram/updates` pipeline knows about telegram stuff, knows how to run the agent pipeline and how to read its result, but and nothing more.

One pipeline is a chain of handlers. A session travels through the chain. The session is a container of data specific to that pipeline pass. Every handler reads the session, adds its own result to it, and hands it on. The first handler that fails stops the chain.

The session carries the state, so the handlers stay small and easy to test. Generics keep the runner and the handler interface common to all pipelines.

## Run it

You need Docker and Docker Compose, a bot token from BotFather, and a public HTTPS URL that Telegram can reach. If behind NAT, use tunnels such as ngrok or cloudflared.

```sh
cp .env.example .env
# fill in TG_TOKEN and PUBLIC_HTTP_HOST
make up
make logs
```
