import {buildQuery} from 'src/timeMachine/utils/queryBuilder'

import {BuilderConfig} from 'src/types'

describe('buildQuery', () => {
  test('single tag', () => {
    const config: BuilderConfig = {
      buckets: ['b0'],
      tags: [{key: '_measurement', values: ['m0']}],
      functions: [],
      aggregateWindow: {period: 'auto'},
    }

    const expected = `from(bucket: "b0")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "m0")`

    const actual = buildQuery(config)

    expect(actual).toEqual(expected)
  })

  test('multiple tags', () => {
    const config: BuilderConfig = {
      buckets: ['b0'],
      tags: [
        {key: '_measurement', values: ['m0', 'm1']},
        {key: '_field', values: ['f0', 'f1']},
      ],
      functions: [],
      aggregateWindow: {period: 'auto'},
    }

    const expected = `from(bucket: "b0")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "m0" or r._measurement == "m1")
  |> filter(fn: (r) => r._field == "f0" or r._field == "f1")`

    const actual = buildQuery(config)

    expect(actual).toEqual(expected)
  })

  test('single tag, multiple functions', () => {
    const config: BuilderConfig = {
      buckets: ['b0'],
      tags: [{key: '_measurement', values: ['m0']}],
      functions: [{name: 'mean'}, {name: 'median'}],
      aggregateWindow: {period: 'auto'},
    }

    const expected = `from(bucket: "b0")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "m0")
  |> aggregateWindow(every: v.windowPeriod, fn: mean)
  |> yield(name: "mean")

from(bucket: "b0")
  |> range(start: v.timeRangeStart, stop: v.timeRangeStop)
  |> filter(fn: (r) => r._measurement == "m0")
  |> aggregateWindow(every: v.windowPeriod, fn: median)
  |> yield(name: "median")`

    const actual = buildQuery(config)

    expect(actual).toEqual(expected)
  })
})
