{
  "schema": "phlame.build.v0",
  "module": {
    "name": "g01/renamed",
    "version": "0.1.0"
  },
  "sources": [
    "counter.phlam",
    "accumulator.phlam",
    "saturating.phlam"
  ],
  "training_plan": "training.json",
  "lock": "phlam.lock",
  "components": [
    {
      "alias": "counter",
      "source": "counter.phlam",
      "class": "Tally",
      "domain": {
        "state_min": -16,
        "state_max": 16,
        "arg_min": -4,
        "arg_max": 4
      },
      "entries": [
        "increment",
        "read"
      ]
    },
    {
      "alias": "accumulator",
      "source": "accumulator.phlam",
      "class": "Reservoir",
      "domain": {
        "state_min": -16,
        "state_max": 16,
        "arg_min": -8,
        "arg_max": 8
      },
      "entries": [
        "add",
        "read"
      ]
    },
    {
      "alias": "saturating",
      "source": "saturating.phlam",
      "class": "Limiter",
      "domain": {
        "state_min": -16,
        "state_max": 16,
        "arg_min": -8,
        "arg_max": 8
      },
      "entries": [
        "add",
        "read"
      ]
    }
  ]
}
