import struct
import sys
import hashlib

def hash(s):
    res = 0x41C64E6D
    for c in s:
        res = (res * 0x3F + ord(c)) & 0xFFFFFFFF
    return res

def compile_lmo(po_path, lmo_path):
    entries = []
    with open(po_path, 'r', encoding='utf-8') as f:
        msgid = ""
        msgstr = ""
        for line in f:
            line = line.strip()
            if line.startswith('msgid "'):
                msgid = line[7:-1]
            elif line.startswith('msgstr "'):
                msgstr = line[8:-1]
                if msgid and msgstr:
                    entries.append((hash(msgid), msgstr.encode('utf-8')))
                    msgid = ""
                    msgstr = ""
    
    entries.sort()
    
    with open(lmo_path, 'wb') as f:
        for h, s in entries:
            f.write(struct.pack('>II', h, len(s)))
            f.write(s)

if __name__ == "__main__":
    compile_lmo(sys.argv[1], sys.argv[2])
