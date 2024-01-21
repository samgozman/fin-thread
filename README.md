# FinThread - News telegram bot powered by AI

![Go](https://img.shields.io/badge/Go-%2300ADD8.svg?logo=go&logoColor=white)
[![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?logo=telegram&logoColor=white)](https://t.me/finthread)
[![Go Doc](https://godoc.org/github.com/?status.svg)](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/)
[![codecov](https://codecov.io/gh/samgozman/fin-thread/graph/badge.svg?token=G8YV1UZC03)](https://codecov.io/gh/samgozman/fin-thread)

Welcome to **FinThread**! FinThread is designed to aggregate financial news from a plethora of sources,
analyze the content using the cutting edge in AI technology, and deliver concise and relevant news
directly to your fingertips through [Telegram](https://t.me/finthread) public channel.

## Features

- **News Aggregator**: Fetches financial news articles from a wide range of sources, ensuring comprehensive news feed.
- **AI-Powered Filtering**: Uses advanced AI models such as Mistral and OpenAI's GPT to filter out unreliable and
  irrelevant news content, focusing on quality and accuracy.
- **Stock Detection**: Identifies stocks that are likely to be affected by the news stories to provide context and
  relevance.
- **AI news Rewriter**: Enhances readability and clarity by rewriting news articles, making them simpler to understand
  while retaining their essential information.
- **Telegram Publishing**: Automatically publishes the refined news to a dedicated Telegram channel for ease of access
  and real-time updates.
- **Economic Calendar Parsing**: Monitors and reports on economic events throughout the week, delivering important
  financial calendar updates.
- **Real-Time Event Tracking**: Stays alert to changes in economic events to provide the channel with the most
  up-to-date information.
- **Summarise Latest News**: Summarises the latest news articles to provide a quick overview of the most important
  events.

## Project Goals

FinThread is not just a bot.
It's an ambitious project aiming to achieve 100% autonomous management of a news channel,
making all necessary decisions independent of human intervention.

## Usage Guide

Here is a quick guide on how to use FinThread if you want to run it yourself.

### Project Structure

The project is split into several entities:

- **[Journalist](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/journalist/)**: Journalists are
  responsible for fetching news from various sources.
  One journalist can have multiple Providers (like RSS feeds).
  Ideally, each journalist should be responsible for his own specific domain (like cryptocurrencies, stocks, etc.).
- **[Composer](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/composer/)**: Composers are
  responsible for composing the news and filtering out irrelevant content using LLMs.
- **[Publisher](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/publisher/)**: Publishers are
  responsible for publishing the news to a specific channel.
- **[Archivist](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/archivist/)**: Archivists are
  responsible for saving the news in a database and retrieving it when needed.
- **[Scavenger](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/scavenger/)**: Scavengers are
  responsible for fetching economic calendar events and other sources that need a custom
  implementation.
- **[Job](https://samgozman.github.io/fin-thread/github.com/samgozman/fin-thread/jobs/)**: Is a set of schedule-based
  tasks that are executed periodically.
  Jobs combine all the above entities to achieve a specific goal.

### Configuration

Configuration is a bit scattered across the project because it's an early stages.
Some things can be configured via ENV, some hardcoded, and some will require you to edit the code directly.
I hope to improve this in the future.

Environment variables are defined in `.env` file. You can copy `.env_example` and fill in the values.

Some things, like flagging words, are hardcoded in the code. You can find them in `config.go` file.

For now, Journalists are defined in the code directly.
The same applies to their settings.
In this demo, I've used two journalists that fetch news from RSS feeds.
Providers for them are defined in `MARKET_JOURNALISTS` and `BROAD_JOURNALISTS` envs in JSON format.

### Running

You can use `docker compose` to run the project locally.

```bash
docker compose up
```

or by running _Makefile_ command:

```bash
make run
```

---

_FinThread is an open-source pet project (proof of concept) and not affiliated with any financial institutions.
The news provided is for informational purposes only and not intended for trading, selling or investing advice._
