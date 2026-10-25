# Historical reference notes

The original archive is intentionally not vendored. See
[ADR 0001](../../docs/decisions/0001-php4-as-reference.md) for inspected files,
licensing observations, hash and decisions.

To reproduce the local research copy from the repository root:

```sh
mkdir -p work
curl -fL https://museum.php.net/php4/php-4.4.9.tar.gz -o work/php-4.4.9.tar.gz
sha256sum work/php-4.4.9.tar.gz
tar -xzf work/php-4.4.9.tar.gz -C work
```

Expected inspected archive SHA-256:
`0c86962c27309a82bfc46d954a29a0d896e469c05f4c5bfd731e33d47c10ad84`.
The archive and extracted tree remain ignored under `work/`. Do not compile or
execute the historical runtime to run the PHLAM tests; no PHP dependency exists.
