# PHLAM language draft v0

Status: bootstrap grammar plus proposed semantic defaults, 2026-10-25. The CLI
checks the syntax described under “Implemented”; it does not execute code,
resolve names, infer effects, parse manifests, or prove semantic contracts.

## Implemented source structure

UTF-8 comments/string contents; ASCII identifiers and keywords are case-sensitive. Source offsets and diagnostic
columns count bytes; line numbers start at one.
Class names start uppercase and contain letters/digits, no underscores; method
names use lowercase snake_case, digits after the first character. Files are code
only, without PHP tags. `///` lines immediately preceding a class provide its
mandatory nonempty purpose. Blank lines and ordinary comments may intervene.
Purpose text is a declared intent, never proof of behavior. Block documentation
comments are not yet a purpose syntax; `/* ... */`, `// ...`, and `# ...` are
ordinary comments.

```ebnf
file        = { newline }, class, { { newline }, class }, { newline } ;
class       = purpose, "class", PascalCase, "{", { member }, "}" ;
member      = "$", identifier, [ "=", expr ], boundary | method ;
method      = "function", snake_case, "(", [ params ], ")", block ;
params      = param, { ",", param }, [ "," ] ;
param       = "$", identifier, [ "=", expr ] ;
block       = "{", { statement }, "}" ;
statement   = "return", [ expr ], boundary
            | "if", "(", expr, ")", block, [ "else", block ]
            | expr, [ ("=" | "+=" | "-="), expr ], boundary ;
expr        = literal | variable | "new", PascalCase, arguments
            | "(", expr, ")" | array
            | ("+" | "-" | "!" | "&"), expr
            | expr, binary_op, expr | expr, "->", identifier
            | expr, "[", expr, "]" | expr, arguments ;
variable    = "$", identifier | "$", variable ;
arguments   = "(", [ expr, { ",", expr }, [ "," ] ], ")" ;
array       = "[", [ item, { ",", item }, [ "," ] ], "]" ;
item        = expr, [ "=>", expr ] ;
literal     = integer | quoted_string | "true" | "false" | "null" ;
```

Layout is omitted from the EBNF except at statement boundaries. Properties may
be uninitialized (`$name`) or have an expression default; semantic restrictions
on defaults remain pending. Duplicate classes, same-kind members, and parameters
are rejected. Required parameters cannot follow optional ones. Integers are
preserved as raw decimal text in the AST; range enforcement awaits semantics.
Strings are single-line quoted tokens with escaped characters, retained raw;
no interpolation or escape interpretation is implemented yet.

Binary precedence, low to high: `||`; `&&`; `== !=`; `< > <= >=`;
`+ - .`; `* / %`. All binary operators associate left. Prefix unary operators
bind tighter, then property/index/call postfix operators. Assignments are
statements, not expressions. Writable targets are variables, indirect variables,
properties, and array items. `&` requires a writable operand. Calls support
ordinary named functions (future imported stdlib), variable call targets, and
static member names. Runtime dynamic member names such as `$obj->$method()` are
planned, not parsed. Arrays use `[]`; PHP `array(...)` is not special syntax.

## Optional semicolons, deterministic layout

Outside expression delimiters, a completed statement ends at `;`, newline, `}`,
or EOF. Two statements on one line need `;`, except the final statement before
`}`. A newline after an incomplete operator/assignment continues the expression.
Newlines inside expression parentheses, call arguments, indexes, and arrays
are layout; class and method braces retain statement boundaries. A newline after
`return` produces a return without a value. Use `return (...)` for a multiline
return expression. Parameter defaults can be parenthesized for multiline layout.
Ordinary block comments preserve their internal newlines.

A completed expression followed on the next line by an operator or postfix
continuation is an ambiguity error P114: `+ - * / % . ( [ -> == != < > <= >=
&& || = += -=`. Parenthesize a continuation, or put `;` before the newline to
make the boundary explicit. This deliberately rejects ambiguous code instead of
guessing intent.

```php
$x = 1 +
     2              // valid: incomplete expression continues
$x = (1
    + 2)            // valid: delimited continuation
$x = 1
    + 2             // rejected: ambiguous completed expression
```

No `for`/`while`/`foreach` parser yet. Traditional for headers will retain mandatory
semicolons even when body statements omit them. This is a planned grammar
extension, not a currently passing feature. Parse nesting is capped at 256
expression levels and 128 blocks; these are parser safeguards, not eval isolation.

## Construction and class access

Only `new ClassName(...)` constructs an object; parentheses are mandatory.
`function init(...)` is the only initializer. Class-named constructors and `__*`
magic methods are rejected, as are `var` and visibility modifiers. All members
are accessible inside an application; external access still requires policy.
Direct `->init(...)` calls are rejected at parse time. Dynamically resolved init
must also be rejected by the future runtime; static rejection is not enforcement
of all dynamic paths.

Proposed semantics: allocate a fresh identity with independent property cells,
evaluate per-instance defaults, call init once, then publish the instance. With
no init, only zero arguments are valid. Initialization failure does not publish
a usable handle. Ordinary assignment copies an object handle, preserving
identity: `$b = $a; $b->increment(2)` changes the same object. `new Counter` twice
creates two independent objects. See pending scenarios in `tests/differential`.
No inheritance, static methods, global application functions, or top-level code.

## Proposed exact value and scope rules (not executed yet)

- Values: null, bool, signed int64, finite float64 (float syntax pending), strings,
  ordered associative arrays, and object handles. Integer overflow is an error.
- No implicit numeric string coercion. Arithmetic requires numeric operands;
  mixed int/float conversion must reject lossy integer conversion. `/` checks
  zero; division details and float determinism need tests before enabling floats.
  `%` requires integers. `.` explicitly concatenates strings only.
- Conditions require booleans; comparisons require compatible types. Equality
  compares type and value; object equality is identity. No PHP loose equality.
- Arrays have integer/string keys without numeric-string key coercion; preserve
  insertion order, replacement retains position. Assignment has value semantics;
  reference implementation may deep-copy arrays before copy-on-write optimization.
  Missing keys/variables/properties are errors; no silent auto-vivification.
- Each call has local variable cells, parameters, and a read-only `$this` binding.
  Ordinary calls do not see caller locals. Argument expressions evaluate left to
  right; normal parameters get values (object handles preserve identity).
- `$alias = &$value` binds the local name to the same writable cell. Later ordinary
  assignment writes the bound cell. Aliasing an undefined variable is an error.
  Array-element alias lifetime and array-copy interactions are unresolved: proposed
  first evaluator reports unsupported instead of inventing copy behavior. Test:
  alias an array element, copy its parent, then mutate each and specify independence.
- `$$name` looks up the string name in the current local scope only; there is no
  arbitrary traversal of host globals. Nonstrings or missing names are errors.
  Dynamic calls resolve only registered application/stdlib symbols in their scope,
  with runtime init restrictions and capability checks. Nested postfix binding of
  indirect variables needs normative examples before language stabilization.
- Failures are typed errors with source span, operation ID and causal stack;
  no warning-and-continue coercion. Resource exhaustion is an explicit error.
  Exception syntax and recovery are unresolved; initial executor aborts invocation.

These defaults intentionally depart from PHP compatibility. Contract generation
must express actual operational relationships and frame conditions, not just
parameter types and natural-language descriptions.

## eval proposal and test obligations

`eval(string)` parses a statement sequence (no PHP tags), not an entire class
file, in an ephemeral overlay sharing the caller's variable cells and `$this`.
It returns the value of an overlay return, or null. It cannot invoke init directly
or indirectly on an existing instance. Class/function definitions in eval are
initially rejected, avoiding implicit persistent symbols.

Budget includes lex/parse/execute, recursion and nested eval. Effective authority
is inherited and may only narrow; no ambient host imports. The frozen learned
artifact and weights cannot change. Overlay failures abort invocation; effects
must remain staged until commit. Concurrency and external nontransactional
adapter behavior require further design. Source spans include overlay ID and
originating eval call.

Required oracle case: `$name = "value"; $$name = 10; $alias = &$value;
eval('$alias += 2'); return $value` must return 12. No runtime for this exists yet.
The learned-execution policy for overlays is explicitly unresolved; an exact
reference overlay may be a disclosed mode but cannot count as learned success.

## Remaining Phase 1 work

Manifest/lock parsing and dependency allowlists, name resolution, defaults/value
validation, operational contract inference, dynamic member syntax, loops, more
complete source diagnostics and error recovery. `check` means bootstrap syntax
and local declaration checks, not deployment readiness.
