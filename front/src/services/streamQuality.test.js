import { test } from 'node:test';
import assert from 'node:assert/strict';
import { selectStreamQuality } from './streamQuality.js';

test('сетка использует low, fullscreen — high, выход возвращает low независимо от порядка качеств', () => {
  const qualities = [{ id: 'high' }, { id: 'low', available: false }];
  assert.equal(selectStreamQuality(qualities, false).id, 'low');
  assert.equal(selectStreamQuality(qualities, true).id, 'high');
  assert.equal(selectStreamQuality(qualities, false).id, 'low');
});

test('без high сохраняется имеющийся поток; у пустой камеры потока нет', () => {
  const low = { id: 'low' };
  assert.equal(selectStreamQuality([low], true), low);
  const custom = { id: 'main' };
  assert.equal(selectStreamQuality([custom], false), custom);
  assert.equal(selectStreamQuality([custom], true), custom);
  assert.equal(selectStreamQuality([], true), undefined);
});
