"""Render static isometric scenery; runtime needs no 3D library.
Art: Kenney Fantasy Town Kit 2.0, CC0, https://kenney.nl/assets/fantasy-town-kit
Download its zip into an empty directory, then:
  python3 -I webapp/assets/make-realm.py /path/kenney_fantasy-town-kit_2.0.zip
Only OBJ geometry and its colour atlas are read. No downloaded code is executed.
"""
import io
import math
import random
import sys
import zipfile
from pathlib import Path
from PIL import Image, ImageDraw

OUT = Path(__file__).resolve().parent
W, H, AA = 1280, 720, 2
SCALE = 31
CACHE = {}
ARCHIVE = zipfile.ZipFile(sys.argv[1])
ATLAS = Image.open(io.BytesIO(ARCHIVE.read('Models/OBJ format/Textures/colormap.png'))).convert('RGB')
LIGHT = (-0.4, 0.85, 0.3)

def project(v):
    x,y,z = v
    return (640+(x-z)*SCALE, 365+(x+z)*SCALE*0.48-y*SCALE*1.35)

def model(name):
    if name in CACHE: return CACHE[name]
    vs,uv,faces=[],[],[]
    for line in ARCHIVE.read('Models/OBJ format/'+name+'.obj').decode().splitlines():
        bits=line.split()
        if not bits: continue
        if bits[0]=='v': vs.append(tuple(map(float,bits[1:4])))
        elif bits[0]=='vt': uv.append(tuple(map(float,bits[1:3])))
        elif bits[0]=='f':
            refs=[p.split('/') for p in bits[1:]]
            verts=[vs[int(p[0])-1] for p in refs]
            coords=[uv[int(p[1])-1] for p in refs if len(p)>1 and p[1]]
            u=sum(p[0] for p in coords)/len(coords) if coords else .5
            v=sum(p[1] for p in coords)/len(coords) if coords else .5
            color=ATLAS.getpixel((min(ATLAS.width-1,max(0,int(u*ATLAS.width))),min(ATLAS.height-1,max(0,int((1-v)*ATLAS.height)))))
            faces.append((verts,color))
    CACHE[name]=faces
    return faces

THEME = "city"

def put(polys,name,x,z,y=0,scale=1,angle=0,tint=None):
    c,s=math.cos(angle),math.sin(angle)
    for verts,color in model(name):
        pts=[(x+(v[0]*c-v[2]*s)*scale,y+v[1]*scale,z+(v[0]*s+v[2]*c)*scale) for v in verts]
        a,b,d=pts[:3]
        u=[b[i]-a[i] for i in range(3)];v=[d[i]-a[i] for i in range(3)]
        n=(u[1]*v[2]-u[2]*v[1],u[2]*v[0]-u[0]*v[2],u[0]*v[1]-u[1]*v[0])
        length=math.sqrt(sum(v*v for v in n)) or 1
        bright=.82+.18*max(0,sum(n[i]*LIGHT[i] for i in range(3))/length)
        if name.startswith('tree') and color[1]>color[0]*1.1:
            if THEME=='nui': color=(205,232,236)
            elif THEME=='nuilua': color=(146,74,62)
            elif THEME=='hang': color=(92,111,148)
            elif THEME=='rung': color=(42,127,76)
        if name.startswith('rock'):
            if THEME=='nui': color=(179,209,229)
            elif THEME=='nuilua': color=(82,65,77)
            elif THEME=='hang': color=(107,103,146)
        if tint and name.startswith(('roof','banner')): color=tint
        color=tuple(min(255,int(v*bright)) for v in color)
        depth=sum(p[0]+p[2]+p[1]*.3 for p in pts)/len(pts)
        polys.append((depth,pts,color))

def house(polys,x,z,width=2.3,height=1.8,tint=(218,82,79),tower=False):
    # Four textured modular walls, facing out, plus a steep roof and a stone doorstep.
    for floor in range(2 if tower else 1):
        for i in range(4):
            put(polys,'wall-wood-door' if i==0 and floor==0 else 'wall-wood-window-glass',x,z,y=floor*width,scale=width,angle=i*math.pi/2,tint=tint)
    top=width*(2 if tower else 1)
    put(polys,'roof-high-point' if tower else 'roof-gable-top',x,z,y=top,scale=width*1.08,tint=tint)
    put(polys,'stairs-stone',x+width*.68,z,scale=width*.7)
    put(polys,'banner-red',x+width*.52,z+.2,y=top*.5,scale=.8,tint=tint)

def base(theme):
    colors={'city':((111,196,229),(224,246,236)), 'vuon':((117,204,232),(222,249,210)), 'rung':((83,159,176),(176,221,181)), 'hang':((66,82,124),(156,169,191)), 'nui':((130,181,222),(238,249,255)), 'nuilua':((122,104,150),(251,184,118))}
    a,b=colors[theme]
    image=Image.new('RGB',(W*AA,H*AA)); draw=ImageDraw.Draw(image)
    for y in range(H*AA):
        f=y/(H*AA);draw.line((0,y,W*AA,y),fill=tuple(int(a[i]*(1-f)+b[i]*f) for i in range(3)))
    # Distant islands and cloud banks, deliberately behind the playable island.
    rng=random.Random(21)
    for _ in range(14):
        x=rng.randrange(-100,W);y=rng.randrange(20,H)
        for dx,dy,r in [(0,0,85),(65,-12,70),(125,4,90)]:
            draw.ellipse(((x+dx-r)*AA,(y+dy-r*.27)*AA,(x+dx+r)*AA,(y+dy+r*.27)*AA),fill=(195,229,232))
    return image

def island(image,theme):
    rng=random.Random(17)
    grass={'city':'#88b84a','vuon':'#9ac95c','rung':'#428d62','hang':'#79899d','nui':'#ddeaf2','nuilua':'#655c64'}[theme]
    rim={'city':'#b8d968','vuon':'#c2e281','rung':'#75b874','hang':'#a0a8b9','nui':'#fcffff','nuilua':'#ad7970'}[theme]
    pts=[]
    for i in range(36):
        t=i*math.tau/36;r=1+rng.uniform(-.045,.045)
        pts.append((640+555*r*math.cos(t),345+245*r*math.sin(t)))
    d=ImageDraw.Draw(image)
    p=lambda a:[(int(x*AA),int(y*AA)) for x,y in a]
    d.ellipse((135*AA,535*AA,1145*AA,664*AA),fill='#a4c6cd')
    for i in range(len(pts)):
        a,b=pts[i],pts[(i+1)%len(pts)]
        if (a[1]+b[1])/2<345:continue
        depth=75+rng.randrange(45)
        d.polygon(p([a,b,(b[0]*.93+44,b[1]+depth),(a[0]*.93+44,a[1]+depth+14)]),fill=['#665b6e','#7c6b78','#8b7882'][i%3])
    d.polygon(p(pts),fill=grass)
    d.line(p(pts+[pts[0]]),fill=rim,width=9*AA)
    # Fine meadow speckle, paths curve through places rather than drawing a square tile grid.
    for _ in range(1000):
        x=rng.uniform(145,1135);y=rng.uniform(130,565)
        if ((x-640)/530)**2+((y-345)/225)**2<.92:
            d.ellipse((x*AA,y*AA,(x+2)*AA,(y+1)*AA),fill=rim)
    return pts

def path(image,points,width=25,color='#d8ca9b'):
    d=ImageDraw.Draw(image);p=[tuple(int(v*AA) for v in project((x,.03,z))) for x,z in points]
    d.line(p,fill='#a49e67',width=(width+8)*AA,joint='curve')
    d.line(p,fill=color,width=width*AA,joint='curve')

def render(theme):
    global THEME
    THEME=theme
    image=base(theme);island(image,theme);polys=[];rng=random.Random(42)
    if theme in ('vuon','rung','nui','nuilua'):
        d=ImageDraw.Draw(image)
        water={'vuon':'#5ebcc5','rung':'#46a8a7','nui':'#9ed7ec','nuilua':'#f18942'}[theme]
        d.ellipse((220*AA,330*AA,400*AA,408*AA),fill=water,outline='#bee2c1',width=4*AA)
    if theme=='hang':
        for x,z in [(1,-5),(3,-6),(-1,-7)]:put(polys,'rock-large',x,z,scale=3.1)
    if theme=='city':
        # coordinates are world-space; the generated manifest places accessible DOM hotspots.
        places=[('pet',-6,3),('hunt',-7,-3),('world',-1,-5.5),('shop',4,-6),('bag',7,-1),('market',6,5),('duel',0,0),('grave',0,8)]
        for _,x,z in places:path(image,[(0,0),(x*.5,z*.5),(x,z)],22)
        for i,(key,x,z) in enumerate(places):
            tint=[(226,102,111),(106,166,101),(115,119,205),(233,168,71),(76,153,189),(215,110,98),(71,169,163),(150,126,192)][i]
            house(polys,x,z,1.9 if key!='duel' else 2.8, tint=tint,tower=key=='world')
            if key=='pet':house(polys,x-1.4,z-.8,1.0,tint=(232,140,154))
            if key=='duel':
                for dx,dz in [(-1.8,-1.8),(1.8,-1.8)]:house(polys,x+dx,z+dz,.75,tint=tint,tower=True)
            if key=='shop':put(polys,'stall-red',x+1.4,z+.8,scale=1.4)
            if key=='grave':
                for dx in [-1.5,-.7,.1]:put(polys,'wall-block-half',x+dx,z+1,scale=.6)
            if key=='hunt':put(polys,'windmill',x+1.3,z,y=3.3,scale=.8)
            if key=='market':
                for dx,dz in [(-1.9,.7),(1.4,1.6)]:put(polys,'stall-green',x+dx,z+dz,scale=1.2)
            if key=='bag':put(polys,'cart',x+1.7,z+1,scale=1.1)
        put(polys,'fountain-round-detail',-1.2,3.5,scale=1.4)
        manifest=[]
        for key,x,z in places:
            sx,sy=project((x,1.4,z));manifest.append({'id':key,'x':round(sx/W*100,2),'y':round(sy/H*100,2)})
        (OUT/'realm-points.json').write_text(__import__('json').dumps(manifest),encoding='utf8')
    else:
        path(image,[(-8,2),(-5,1),(-2,3),(1,1),(4,-1),(6,-4)],28,color='#d6ccab' if theme!='nuilua' else '#d68d66')
        house(polys,-7,2,width=1.8,tint=(118,161,189))
        for i in range(22):
            x,z=rng.uniform(-8,8),rng.uniform(-6,6)
            if abs(z-(1-x*.4))<1.5:continue
            if theme in ('vuon','rung'):
                put(polys,'tree-high-round' if i%3 else 'tree',x,z,scale=rng.uniform(.8,1.4),tint=None)
            else:
                put(polys,'rock-large' if i%2 else 'rock-wide',x,z,scale=rng.uniform(1,2.4))
                if i%4==0:put(polys,'tree',x,z,scale=.9)
        # The chapter boss landmark, visibly larger than the small stage camps.
        house(polys,6,-4,width=2.4,tower=True,tint={'vuon':(207,153,73),'rung':(69,137,115),'hang':(128,105,177),'nui':(104,177,218),'nuilua':(217,91,64)}[theme])
        if theme in ('vuon','rung'):put(polys,'watermill',-6,-2,y=1.2,scale=1.3)
    # Border foliage keeps the centre readable and gives the city a frame.
    for i in range(34):
        t=i*math.tau/34;x,z=11.5*math.cos(t),8.6*math.sin(t)
        if (x-z)**2>270 or x+z < -9:continue
        put(polys,'tree' if i%3 else 'tree-high-round',x,z,scale=.8+rng.random()*.5)
        if i%4==0:put(polys,'rock-wide',x+.5,z,scale=.75)
    draw=ImageDraw.Draw(image)
    for _,verts,color in sorted(polys,key=lambda p:p[0]):
        draw.polygon([(round(project(v)[0]*AA),round(project(v)[1]*AA)) for v in verts],fill=color)
    image=image.resize((W,H),Image.Resampling.LANCZOS)
    image.save(OUT/('realm-'+theme+'.png'),optimize=True)
    image.save(OUT/('realm-'+theme+'.webp'),'WEBP',quality=88,method=6)
    print(theme,len(polys))

for theme in ['city','vuon','rung','hang','nui','nuilua']:render(theme)
(OUT/'FANTASY-TOWN-LICENSE.txt').write_bytes(ARCHIVE.read('License.txt'))
