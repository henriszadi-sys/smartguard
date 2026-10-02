"""Conversion Markdown -> PDF des documents SmartGUARD (tableaux, titres, listes).

Usage : python3 md2pdf.py <source.md> <sortie.pdf> "<titre de pied de page>" [paysage]
Dépendance : reportlab (pip install reportlab).
"""
import re
import sys
from reportlab.lib.pagesizes import A4 as _A4, landscape
A4=landscape(_A4) if len(sys.argv)>4 and sys.argv[4]=='paysage' else _A4
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer, Table, TableStyle, KeepTogether
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib import colors
from reportlab.lib.units import cm
NAVY=colors.HexColor("#1F3864"); GREY=colors.HexColor("#F2F2F2")
st=getSampleStyleSheet()
S={'h1':ParagraphStyle('h1',parent=st['Title'],textColor=NAVY,fontSize=20,spaceAfter=6),
 'h2':ParagraphStyle('h2',parent=st['Heading2'],textColor=NAVY,fontSize=14,spaceBefore=12,spaceAfter=4,keepWithNext=1),
 'h3':ParagraphStyle('h3',parent=st['Heading3'],textColor=NAVY,fontSize=11.5,spaceBefore=8,spaceAfter=3,keepWithNext=1),
 'p':ParagraphStyle('p',parent=st['Normal'],fontSize=9.5,leading=12.5,spaceAfter=4),
 'li':ParagraphStyle('li',parent=st['Normal'],fontSize=9.5,leading=12.5,leftIndent=12,bulletIndent=2,spaceAfter=2),
 'td':ParagraphStyle('td',parent=st['Normal'],fontSize=8,leading=10),
 'th':ParagraphStyle('th',parent=st['Normal'],fontSize=8,leading=10,textColor=colors.white,fontName='Helvetica-Bold')}
def inl(t):
    t=t.replace('&','&amp;').replace('<','&lt;').replace('>','&gt;')
    t=re.sub(r'\*\*(.+?)\*\*',r'<b>\1</b>',t)
    t=re.sub(r'(?<![\w*])\*(?!\s)(.+?)(?<!\s)\*(?!\w)',r'<i>\1</i>',t)
    t=re.sub(r'`(.+?)`',r'<font face="Courier">\1</font>',t)
    t=t.replace('\\*','*')
    return t
W=A4[0]-3.6*cm
def table(rows):
    rows=[[c.strip() for c in r.strip().strip('|').split('|')] for r in rows]
    rows=[r for r in rows if not all(set(c)<=set('-: ') for c in r)]
    n=len(rows[0]); 
    lens=[max(len(r[i]) if i<len(r) else 0 for r in rows) for i in range(n)]
    words=[max((len(w) for r in rows if i<len(r) for w in r[i].replace('*','').split()),default=4) for i in range(n)]
    mins=[w*5.0+10 for w in words]
    for i,h in enumerate(rows[0]):
        if h.startswith('Résultat machine'): mins[i]=150
        if h.startswith('Statut final'): mins[i]=70
    lens=[min(max(l,8),45) for l in lens]
    rest=W-sum(mins)
    if rest<0: widths=[W*m/sum(mins) for m in mins]
    else:
        extra=[max(l*4.4-m,0) for l,m in zip(lens,mins)]; te=sum(extra) or 1
        widths=[m+rest*e/te for m,e in zip(mins,extra)]
    data=[[Paragraph(inl(c),S['th'] if k==0 else S['td']) for c in r]+[Paragraph('',S['td'])]*(n-len(r)) for k,r in enumerate(rows)]
    t=Table(data,colWidths=widths,repeatRows=1)
    t.setStyle(TableStyle([('BACKGROUND',(0,0),(-1,0),NAVY),('GRID',(0,0),(-1,-1),0.4,colors.HexColor("#BFBFBF")),
      ('VALIGN',(0,0),(-1,-1),'TOP'),('ROWBACKGROUNDS',(0,1),(-1,-1),[colors.white,GREY]),
      ('TOPPADDING',(0,0),(-1,-1),3),('BOTTOMPADDING',(0,0),(-1,-1),3),('LEFTPADDING',(0,0),(-1,-1),4),('RIGHTPADDING',(0,0),(-1,-1),4)]))
    return t
def build(src,out,title):
    lines=open(src,encoding='utf-8').read().split('\n'); story=[]; i=0
    while i<len(lines):
        l=lines[i]
        if l.startswith('|'):
            tb=[]
            while i<len(lines) and lines[i].startswith('|'): tb.append(lines[i]); i+=1
            story+= [table(tb),Spacer(1,5)]; continue
        if l.strip()=='---': story.append(Spacer(1,4))
        elif l.startswith('### '): story.append(Paragraph(inl(l[4:]),S['h3']))
        elif l.startswith('## '): story.append(Paragraph(inl(l[3:]),S['h2']))
        elif l.startswith('# '): story.append(Paragraph(inl(l[2:]),S['h1']))
        elif re.match(r'^\s*- ',l): story.append(Paragraph(inl(re.sub(r'^\s*- ','',l)),S['li'],bulletText='•'))
        elif re.match(r'^\d+\. ',l):
            m=re.match(r'^(\d+)\. (.*)',l); story.append(Paragraph(inl(m.group(2)),S['li'],bulletText=m.group(1)+'.'))
        elif l.strip(): story.append(Paragraph(inl(l),S['p']))
        i+=1
    def foot(c,d):
        c.saveState(); c.setFont('Helvetica',7.5); c.setFillColor(colors.grey)
        c.drawString(1.8*cm,1.1*cm,title); c.drawRightString(A4[0]-1.8*cm,1.1*cm,f"Page {d.page}"); c.restoreState()
    SimpleDocTemplate(out,pagesize=A4,leftMargin=1.8*cm,rightMargin=1.8*cm,topMargin=1.6*cm,bottomMargin=1.8*cm,title=title,author="Labenie").build(story,onFirstPage=foot,onLaterPages=foot)
import sys; build(sys.argv[1],sys.argv[2],sys.argv[3])
