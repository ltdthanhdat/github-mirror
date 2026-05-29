## 1. Update mirror creation contract

- [x] 1.1 Replace source/target owner and repository inputs in the mirror creation form with source and target GitHub URL inputs.
- [x] 1.2 Update the mirror creation handler to read `source_url` and `target_url`, parse owner/repository values, and normalize clone URLs before saving.
- [x] 1.3 Add request validation for unsupported GitHub URL shapes and return invalid-request errors without creating a mirror.

## 2. Verify behavior

- [x] 2.1 Update existing tests to use the URL-based mirror creation contract.
- [x] 2.2 Add coverage for invalid repository URL submissions to confirm the handler rejects them and does not persist a mirror configuration.
