# Веб-фронтенд Task Board

> Kanban-приложение для организации задач.

Веб-фронтенд Task Board, написан на Vue.js 3 + TypeScript + Vite.

Общую информацию о проекте смотри в корневом README репозитория.

## Настройка проекта

```shell
pnpm install
```

### Разработка

#### Указать backend-сервер

Можно разрабатывать фронтенд против любого доступного backend. Для этого нужно задать переменную окружения `DEV_PROXY`.

Рекомендуемый способ:

- Скопируй `.env.local.example` как `.env.local`
- Раскомментируй строку `DEV_PROXY`
- Укажи URL backend, который хочешь использовать

Например:

```shell
DEV_PROXY=http://192.168.1.135:19876
```

#### Запустить dev-сервер (компиляция и hot-reload)

```shell
pnpm run dev
```

Dev-сервер поднимается на `http://127.0.0.1:4173`.

### Сборка для production

```shell
pnpm run build
```

### Линтер и автофикс

```shell
pnpm run lint
pnpm run lint:fix
```
