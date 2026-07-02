# Local LLM Distillation

Local LLM is optional. Without it, `cc-session handoff` writes deterministic filtered artifacts and an evidence index.

When enabled, the endpoint must be OpenAI-compatible:

```json
{
  "local_llm": {
    "enabled": true,
    "base_url": "http://127.0.0.1:8000/v1",
    "api_key": "",
    "model": "Qwen3.6-35B-A3B",
    "max_context": 32000,
    "max_output_tokens": 4096,
    "timeout_seconds": 120,
    "min_filtered_chars": 8000,
    "temperature": 0,
    "top_p": 0.95,
    "top_k": 20
  }
}
```

`--llm auto` uses filtered output for short sessions and calls the LLM only when redacted filtered chars meet `min_filtered_chars`. `--llm never` disables LLM. `--llm always` requires Local LLM config.

Default temperature is `0` to keep handoffs deterministic. `top_p`, `top_k`, max context, and max output tokens are explicit so vLLM/OpenAI-compatible servers do not silently choose uncontrolled defaults.
