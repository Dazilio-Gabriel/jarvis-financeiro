"""Gera o PDF de um relatorio em Markdown (docs/entregas/*.md) usando o Edge em modo headless.

Uso:  python scripts/relatorio-pdf.py docs/entregas/C1-relatorio.md
Requer: pip install markdown  (o Edge ja vem com o Windows).
O PDF sai ao lado do .md, com o mesmo nome.
"""
import io, re, sys, subprocess, pathlib, tempfile, time, markdown

src = pathlib.Path(sys.argv[1]).resolve()
out_pdf = src.with_suffix('.pdf')
out_html = pathlib.Path(tempfile.gettempdir()) / (src.stem + '.html')
body = markdown.markdown(io.open(src, encoding='utf-8').read(),
                         extensions=['tables', 'fenced_code', 'sane_lists', 'toc'])
# Tabela chave/valor do Markdown exige cabecalho; se ele for vazio, some do PDF.
body = re.sub(r'<thead>\s*<tr>(?:\s*<th>\s*</th>)+\s*</tr>\s*</thead>', '', body)
css = """
@page { size: A4; margin: 22mm 18mm 20mm 18mm; }
body { font-family: 'Segoe UI', Arial, sans-serif; font-size: 10.5pt; line-height: 1.45; color: #1a1a1a; }
h1 { font-size: 20pt; margin: 0 0 6pt; color: #0b2545; border-bottom: 2px solid #0b2545; padding-bottom: 4pt; }
h2 { font-size: 14pt; color: #0b2545; margin: 18pt 0 6pt; page-break-after: avoid; }
h3 { font-size: 11.5pt; color: #13315c; margin: 12pt 0 4pt; page-break-after: avoid; }
p { margin: 0 0 7pt; text-align: justify; }
table { border-collapse: collapse; width: 100%; margin: 6pt 0 10pt; font-size: 9pt; }
th, td { border: 1px solid #b8c0cc; padding: 3pt 5pt; vertical-align: top; text-align: left; }
th { background: #e8eef7; }
tr { page-break-inside: avoid; }
code { font-family: Consolas, 'Courier New', monospace; font-size: 9pt; background: #f2f4f7; padding: 0 2pt; }
pre { background: #f2f4f7; border: 1px solid #d5dbe5; padding: 6pt 8pt; font-size: 8.5pt; line-height: 1.3; white-space: pre-wrap; page-break-inside: avoid; }
pre code { background: none; padding: 0; }
blockquote { border-left: 3px solid #b8c0cc; margin: 6pt 0; padding: 2pt 10pt; color: #444; }
hr { border: 0; border-top: 1px solid #b8c0cc; margin: 14pt 0; }
a { color: #0b4f9c; text-decoration: none; word-break: break-all; }
ul, ol { margin: 0 0 7pt 18pt; padding: 0; }
"""
io.open(out_html, 'w', encoding='utf-8').write(
    f"<!doctype html><html lang='pt-BR'><head><meta charset='utf-8'><title>{src.stem}</title>"
    f"<style>{css}</style></head><body>{body}</body></html>")

edge = r"C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe"
profile = pathlib.Path(tempfile.gettempdir()) / 'edge-headless-relatorio'
if out_pdf.exists():
    out_pdf.unlink()
subprocess.run([edge, "--headless=new", "--disable-gpu", "--no-first-run", "--no-pdf-header-footer",
                f"--user-data-dir={profile}", f"--print-to-pdf={out_pdf}", out_html.as_uri()],
               timeout=120, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
for _ in range(30):
    if out_pdf.exists() and out_pdf.stat().st_size > 1000:
        break
    time.sleep(1)
print("gerado:" if out_pdf.exists() else "FALHOU:", out_pdf)
