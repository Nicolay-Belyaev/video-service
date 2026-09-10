import { test } from 'node:test';
import assert from 'node:assert/strict';
import { retryConnection } from './retryConnection.js';

test('первая попытка и ровно три повтора с интервалом три секунды', (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] });
  let attempts = 0;
  let callbacks;
  const statuses = [];
  const stop = retryConnection(
    (events) => {
      attempts++;
      callbacks = events;
      return () => {};
    },
    (status) => statuses.push(status),
  );

  assert.equal(attempts, 1);
  for (let retry = 1; retry <= 3; retry++) {
    callbacks.failed(new Error('Offline'));
    // Повторные события браузера не должны создавать дополнительные таймеры.
    callbacks.failed(new Error('Offline'));
    context.mock.timers.tick(2999);
    assert.equal(attempts, retry);
    context.mock.timers.tick(1);
    assert.equal(attempts, retry + 1);
  }
  callbacks.failed(new Error('Offline'));
  context.mock.timers.tick(30000);
  assert.equal(attempts, 4);
  assert.match(statuses.at(-1), /после 3 повторных попыток/);
  stop();
});

test('остановка отменяет повтор и игнорирует запоздалые события', (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] });
  let attempts = 0;
  let callbacks;
  const stop = retryConnection(
    (events) => {
      attempts++;
      callbacks = events;
      return () => {};
    },
    () => {},
  );
  callbacks.failed(new Error('Offline'));
  stop();
  callbacks.connected();
  callbacks.failed(new Error('Late event'));
  context.mock.timers.tick(30000);
  assert.equal(attempts, 1);
});

test('успешное подключение сбрасывает счётчик повторов', (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] });
  let callbacks;
  const statuses = [];
  const stop = retryConnection(
    (events) => {
      callbacks = events;
      return () => {};
    },
    (status) => statuses.push(status),
  );
  callbacks.failed(new Error('Offline'));
  context.mock.timers.tick(3000);
  callbacks.connected();
  callbacks.failed(new Error('Disconnected'));
  assert.match(statuses.at(-1), /1\/3/);
  stop();
});

test('исчерпание повторов однократно возвращает плеер к ручному запуску', (context) => {
  context.mock.timers.enable({ apis: ['setTimeout'] });
  let callbacks;
  let exhausted = 0;
  const stop = retryConnection(
    (events) => {
      callbacks = events;
      return () => {};
    },
    () => {},
    () => {
      exhausted++;
    },
  );
  for (let attempt = 0; attempt < 4; attempt++) {
    callbacks.failed(new Error('Offline'));
    if (attempt < 3) {
      assert.equal(exhausted, 0);
      context.mock.timers.tick(3000);
    }
  }
  callbacks.failed(new Error('Duplicate failure'));
  assert.equal(exhausted, 1);
  stop();
});
