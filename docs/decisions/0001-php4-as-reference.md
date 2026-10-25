# ADR 0001: PHP 4 is historical reference, not the execution substrate

Decision: accepted for bootstrap, 2026-10-25.

## Evidence acquired

Inspected the official [PHP 4.4.9 archive](https://museum.php.net/php4/php-4.4.9.tar.gz),
retrieved on 2026-10-25. The local research copy is `work/php-4.4.9/`, ignored by
Git. Download SHA-256:
`0c86962c27309a82bfc46d954a29a0d896e469c05f4c5bfd731e33d47c10ad84`.
This records the bytes inspected; it is not a verified release signature.
The [release announcement](https://www.php.net/releases/4_4_9.php) identifies this
as the last PHP 4.4 release. No historical interpreter was built or executed.

Paths/line numbers below are relative to the extracted `php-4.4.9/` archive:

| File and location inspected | Observation | Relevance to PHLAM |
| --- | --- | --- |
| `Zend/zend_language_parser.y:229-234,381-399` | Grammar actions call `zend_do_*` for function/class/property declarations; properties use T_VAR | Study familiar syntax; write separate AST construction and require class purposes |
| `Zend/zend_language_parser.y:425-426,716` | new-object actions and eval production call compiler helpers | Preserve dynamic features as explicit language obligations |
| `Zend/zend_language_scanner.l:1277-1341,1391-1453` | PHP tag modes, variable tokenization, comments, close tag as implicit boundary | Remove mixed HTML/tag modes; define newline boundaries afresh |
| `Zend/zend_compile.c:756,1159-1188,1599-1690` | Function/class compilation and inherited constructor machinery | Avoid historical constructor conventions; choose new + init |
| `Zend/zend.h:278-279,508-575` | zval contains reference flags/counts and separation macros | Understand aliasing and copying questions; specify cells/identity independently |
| `Zend/zend_execute_API.c:546-575,610-665` | Active symbol-table handling; eval compiles a string and executes a temporary op array | Caller-scope mutation needs explicit tests; old execution machinery supplies no new capability boundary |
| `tests/classes/class_example.phpt` | Object assignment/mutation examples with expected output | Inform tests for deliberately chosen identity semantics, not blanket compatibility |
| `tests/lang/024.phpt:15-50` | Assignments using dynamically computed variable names | Inform fresh $$name scenarios |
| `run-tests.php:1-27` | Historical test-runner header and provenance | Use a small fresh Go corpus harness instead of porting the runner |

The [AST adoption RFC](https://wiki.php.net/rfc/abstract_syntax_tree) describes the
later move from parser actions emitting opcodes to parsing an AST before
compilation. It explains why decoupling makes grammar/compiler maintenance easier.
PHLAM adopts the separation as compiler/training infrastructure. The RFC does not
establish feasibility of learned execution. The [current eval manual](https://www.php.net/manual/en/function.eval.php)
is a scope reference, not authoritative evidence for all PHP 4 details or a
PHLAM sandbox design.

## Reuse and incompatibilities

Reuse ideas and test questions: variable/name distinction, property access,
source-aware diagnostics, call scopes and alias identity. Write the lexer,
recursive-descent parser, AST and tests from scratch. No PHP/Zend implementation,
generated scanner/parser, test text, or host runtime is included in PHLAM source.

PHLAM deliberately changes tags, class documentation, top-level execution,
properties, constructors, semicolon rules, naming, object identity defaults and
coercions. Zend's conventional opcode execution is useful historical reference,
not the target. A graph IR can scaffold the oracle/training pipeline. The deployed
application must depend on frozen learned parameters under exact safeguards.

## File-specific license observations

This is an inventory of inspected notices, not clearance to copy arbitrary files.
[PHP's licensing page](https://www.php.net/license/) is context; the archived file
headers control this inventory.

- Archive `LICENSE` states PHP License 3.01. `main/main.c:1-20` and
  `run-tests.php:1-27` identify that license in their headers.
- `Zend/LICENSE` states Zend Engine License 2.00. Inspected headers in
  `zend_language_parser.y`, `zend_language_scanner.l`, `zend_compile.c`,
  `zend_execute.c`, `zend_execute_API.c`, and `zend.h` identify Zend 2.00.
- `Zend/zend_language_parser.c:1-35` includes the GNU Bison 2.3 generated skeleton
  notice: GPL terms with a stated special exception. The generated file also
  embeds Zend grammar material. Do not assume one root license covers it.
- The inspected `.phpt` examples do not carry individual license headers; this
  mission does not copy them or claim their independent license is established.
- Bundled extensions/libraries and uninspected generated files are not audited.
  Any future copied file needs its own notice/provenance assessment first.

License selection for original PHLAM work is deferred to the project owner.
No third-party implementation has been vendored, so this mission creates no
mixed Zend/PHP runtime dependency.
