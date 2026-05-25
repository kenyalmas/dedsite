package handlers

import (
	"math/rand"
	"net/http"
)

type NotFoundPage struct {
	Path      string
	Art       string
	Remark    string
	ErrnoCode string
	Title     string
}

var notFoundVariants = []NotFoundVariant{
	{
		Art: ` 
  /\___/\      /\___/\
 (  o o  )    (  o o  )
  >  ^  < \__/ >  ^  <
 /     \        /     \ 
/__ __  \_/__\_/  __ __\`,
		Remark:    "this route was audited by cats and flagged as purely decorative.",
		ErrnoCode: "ENOENT",
		Title:     "404: page fault in user space",
	},
	{
		Art: `
   /\_/\      /\_/\  
  ( -.- )    ( o.o ) 
  / >^< \    / >^< \ 
 /_/___\_\  /_/___\_\ 
                    `,
		Remark:    "the scheduler is up, but the workers joined the writers strike.",
		ErrnoCode: "EFAULT",
		Title:     "404: bus error on route lookup",
	},
	{
		Art: `       
	   /\_/\      
    .-( o.o )-.   
   / (  =^=  )\  
  / .-""""""-. \ 
 | /  .--.    \ |
 | \ (____)   / |
  \ '-.____.-' / 
   '._  __  _.'
      \/  \/`,
		Remark:    "the cat ate your cookies.",
		ErrnoCode: "SIGSEGV",
		Title:     "404: kernel panic in route table",
	},
	{
		Art: `     
	   /\_/\      
 /\_/\( -.- )/\_/\   
( o.o )     ( o.o )
 > ^ < /   \ > ^ < 
|_/|__/_|_|_\__|\_|  `,
		Remark:    "the cat ate my cache.",
		ErrnoCode: "ENXIO",
		Title:     "404: device not purrmitted",
	},
	{
		Art: ` 
 /\___/\      /\___/\
(  o.o  )\   /(  -.- )
  > ^ <   \_/   > ^ <
 /|\___/  __  \___/|\  
/_|___/__/  \__\___|_\ `,
		Remark:    "the night shift asked Cluade for a fix.",
		ErrnoCode: "ETIMEDOUT",
		Title:     "404: timeout in nap cycle",
	},
	{
		Art: `  
  /^ ^\
 / 0 0 \
 V\ Y /V
  / - \
 | |   \ /
 | |( __V`,
		Remark:    "our process got the zoomies and sprinted past your endpoint.",
		ErrnoCode: "EWOULDBLOCK",
		Title:     "404: nonblocking tail chase",
	},
	{
		Art: `   
   /^ ^\
  / o o \
  V\ Y /V
   / - \
  /|   |\
 (__| |__)`,
		Remark:    "that route got sniffed, approved, then forgotten immediately.",
		ErrnoCode: "ESRCH",
		Title:     "404: good boy lost target",
	},
	{
		Art: `  
  / \__
 (    @\___
 /         O
/   (_____/
/_____/   U`,
		Remark:    "the watchdog fetched the packet and buried it in another subnet.",
		ErrnoCode: "ENETDOWN",
		Title:     "404: route chased by watchdog",
	},
	{
		Art: `   
   /^ ^\
  / o o \
  V\ Y /V
   / - \
  /|   |\
 (__|_|__)`,
		Remark:    "the watchdog is sleeping on the back porch.",
		ErrnoCode: "EOVERFLOW",
		Title:     "404: overflow in aquarium buffer",
	},
	{
		Art: `
			><(((('>
      ><(((('>
><(((('>`,
		Remark:    "the request joined a school and migrated southbound.",
		ErrnoCode: "ENOLINK",
		Title:     "404: missing school link",
	},
	{
		Art: `    
	><(((°>
 ><((('>   ><('>
     ><(((°>
  ><(('>
><(((°>   ><(°>`,
		Remark:    "the endpoint took the bait, then swam off with the socket.",
		ErrnoCode: "EIO",
		Title:     "404: fishhook I/O anomaly",
	},
	{
		Art: `  
	  ><(°>
<°><    ><>
    ><>
<°)><    <°><
    ><>
<°><   <><`,
		Remark:    "tiny packet fish nibbled the URL down to a NULL byte.",
		ErrnoCode: "EMSGSIZE",
		Title:     "404: message too snackable",
	},
}

type NotFoundVariant struct {
	Art       string
	Remark    string
	ErrnoCode string
	Title     string
}

func (h Handler) NotFound(w http.ResponseWriter, r *http.Request) {
	variant := notFoundVariants[rand.Intn(len(notFoundVariants))]
	w.WriteHeader(http.StatusNotFound)
	h.render(w, "404.html", NotFoundPage{
		Path:      r.URL.Path,
		Art:       variant.Art,
		Remark:    variant.Remark,
		ErrnoCode: variant.ErrnoCode,
		Title:     variant.Title,
	})
}
