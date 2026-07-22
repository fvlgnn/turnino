# Legacy migration notes

The PHP applications under `.archivio/legacy/` are private historical references, not modules to port line by line.

Preserve these behaviors:

- immediate browser-based counter registration;
- automatic and manual calls;
- public list of recent calls with sound and emphasis;
- dedicated display for a counter;
- small administration surface;
- configurable numbering and presentation.

Do not carry these implementation details forward:

- browser sessions used as permanent counter identity;
- deletion of counters and call history when sessions expire;
- polling that reloads complete pages;
- plaintext administrator secrets;
- query construction from untrusted input;
- MyISAM tables without transactions or foreign keys;
- demo behavior embedded in production rendering;
- cleanup work performed opportunistically during page requests.

If legacy material is ever published separately, replace `demo.sql` with generated fictional data and inspect every asset and dependency for redistribution rights. Existing restrictive copyright headers remain effective unless the copyright holder explicitly relicenses those files.
