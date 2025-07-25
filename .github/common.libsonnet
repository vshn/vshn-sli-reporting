local formatLabels = function(labels)
  local lf = std.join(', ', std.map(function(l) '%s="%s"' % [l, labels[l]], std.objectFields(labels)));
  '{%s}' % [lf];

// returns a series object with correctly formatted labels.
// labels can be modified post creation using `_labels`.
local series = function(name, labels, values) {
  _name:: name,
  _labels:: labels,
  series: self._name + formatLabels(self._labels),
  values: values,
};

// returns a test object with the given series and samples. Sample interval is 30s
// the evaluation time is set one hour in the future since all our queries operate on a 1h window
local test = function(name, series, query, samples, interval='30s', eval_time='1h') {
  name: name,
  interval: interval,
  input_series: if std.isArray(series) then series else std.objectValues(series),
  promql_expr_test: [
    {
      expr: query,
      eval_time: eval_time,
      exp_samples: if std.isArray(samples) then samples else [samples],
    },
  ],
};

{
  series: series,
  formatLabels: formatLabels,
  test: test,
}
