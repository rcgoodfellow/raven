/******************************************************************************
 *
 * iamme - A simple program to send a dhcp request to a dhcp server and do
 *         nothing else. This program is for usefull for machines in a dynamic
 *         dns network that report their name-address binding through dhcp
 *         e.g., a dnsmasq setup.
 *
 *         This program takes a interface name and a dhcp server address as
 *         parameters. It then sends a dhcp request with the hostname option
 *         set to whatever the hostname of the system is at time time to the
 *         specified server. If the server is running a coupled dhcp/dns
 *         service, the new hostname-address binding will now be visible to
 *         the rest of the network.
 *
 *         Why not just use dhclient -x or dhclient -r, in my experience both
 *         of these can cause transient connectivity issues that can screw
 *         up automation in progress. This program does not change a thing
 *         about your live connections.
 *
 *****************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <unistd.h>
#include <sys/param.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <ifaddrs.h>

#define DHCP_CHADDR_LEN    16
#define DHCP_SNAME_LEN     64
#define DHCP_FILE_LEN      128
#define SYSFS_ETHADDR_LEN  17

#define DHCP_OPTYPE_REQUEST 1
#define DHCP_HWTYPE_ETHERNET 1
#define DHCP_HWETH_LEN 6
#define DHCP_IAMME_TXN_ID 0x1701D

#define DHCP_OPT_REQUESTTYPE 53
#define DHCP_OPT_HOSTNAME 12
#define DHCP_OPT_COOKIE_SIZE 4

#define DHCP_SERVER_PORT 67

#define SUCCESS 0
#define FAILURE 1

// 
// Data structures
//

struct me_opts {
  char cookie[DHCP_OPT_COOKIE_SIZE];             //magic cookie option
  char type_code, type_len, type;                   //request type option
  char name_code, name_len, name[HOST_NAME_MAX];    //hostname option
  char end;                                         //end option
} __attribute__((packed));

struct dhcp_pkt
{
  char 
    op, //opcode 1 = REQUEST, 2 = REPLY
    htype,  //hardware address type 1 = ethernet
    hlen, // hardware address length (6 for ethernet)
    hops; // = 0 for client

  uint32_t 
    xid; // transaction id

  uint16_t
    secs, //seconds ellapesd since client began address acquisition process
    flags; // 0x8000 for broadcast 0x0 otherwise

  uint32_t
    ciaddr, //client ip address || client -> server
    yiaddr, //your ip address   || server -> client
    siaddr, //bootstrap server address
    giaddr; //relay agent address

  char 
    chaddr[DHCP_CHADDR_LEN], //client hardware address
    sname[DHCP_SNAME_LEN], //server host name
    file[DHCP_FILE_LEN]; //boot file name

  struct me_opts 
    options; //my options
} __attribute__((packed));

//
// Forward declarations
//

void usage();
int parseArgs(int argc, char **argv, char **interface, char **server);
void set_chaddr(struct dhcp_pkt *pkt, char *interface);
void set_ciaddr(struct dhcp_pkt *pkt);
void set_name(struct dhcp_pkt *pkt);

//
// Entry point
//

int main(int argc, char **argv)
{
  char *interface, *dhcp_server;
  int err = parseArgs(argc, argv, &interface, &dhcp_server);
  if (err) {
    usage();
    exit(1);
  }
  //create the packet to tell the dhcp server our hostname
  struct dhcp_pkt pkt = {
    .op =     DHCP_OPTYPE_REQUEST,
    .htype =  DHCP_HWTYPE_ETHERNET,
    .hlen =   DHCP_HWETH_LEN,
    .hops =   0,

    .xid =    htonl(DHCP_IAMME_TXN_ID),

    .secs =   0,
    .flags =  0,

    .ciaddr = 0,
    .yiaddr = 0,
    .siaddr = 0,
    .giaddr = 0,
    .options = {
      .cookie = {0x63, 0x82, 0x53, 0x63},
      .type_code = DHCP_OPT_REQUESTTYPE,
      .type_len = 1,
      .type = 3,
      .name_code = DHCP_OPT_HOSTNAME,
    }
  };

  set_chaddr(&pkt, interface);  //set the hardware address
  set_ciaddr(&pkt);             //set the ip address
  set_name(&pkt);               //set the hostname

  //don't need the server name or filename so just zero them out
  memset(pkt.sname, 0, DHCP_SNAME_LEN);
  memset(pkt.file, 0, DHCP_FILE_LEN);

  //send the request
  int sock = socket(AF_INET, SOCK_DGRAM, 0);
  struct sockaddr_in server = {
    .sin_addr = { .s_addr = 0 },
    .sin_family = AF_INET,
    .sin_port = htons(DHCP_SERVER_PORT),
    .sin_zero = 0
  };
  inet_aton(dhcp_server, (struct in_addr *)&server.sin_addr.s_addr);
  sendto(sock, &pkt, sizeof(pkt), 0, (const struct sockaddr*)&server, sizeof(server));
  close(sock);
  
}

//
// Helper functions
//

void usage()
{
  printf("usage:\n  iamme interface dhcp-server\n");
}

int parseArgs(int argc, char **argv, char **interface, char **server)
{
  if(argc < 3) {
    return FAILURE;
  }

  *interface = argv[1];
  *server = argv[2];

  return SUCCESS;
}

void set_chaddr(struct dhcp_pkt *pkt, char *interface)
{
  //read the mac address of the device in question from sysfs
  char *fmt = "/sys/class/net/%s/address";
  int path_sz = snprintf(NULL, 0, fmt, interface);
  char *path = malloc(path_sz+1);
  snprintf(path, path_sz+1, fmt, interface);

  FILE *f = fopen(path, "r");
  if (!f) {
    printf("error opening sysfs address!\n");
    printf("path: %s\n", path);
    exit(EXIT_FAILURE);
  }
  free(path);

  char buf[SYSFS_ETHADDR_LEN];
  int n = fread(buf, sizeof(char), SYSFS_ETHADDR_LEN, f);
  if (n < SYSFS_ETHADDR_LEN) {
    printf("error reading eth0 address\n");
    exit(EXIT_FAILURE);
  }

  //copy the mac address into the dhcp pkt
  memset(pkt->chaddr, 0, DHCP_CHADDR_LEN);
  for(int i=0,j=0; i<SYSFS_ETHADDR_LEN; i+=3, j+=sizeof(uint8_t)) {

    uint8_t x = (uint8_t)strtoul(&buf[i], NULL, 16);
    memcpy(&pkt->chaddr[j], &x, sizeof(uint8_t));

  }

  //DEBUG
  printf("mac: ");
  for(size_t i=0; i<DHCP_HWETH_LEN*sizeof(uint8_t); i+= sizeof(uint8_t)) {
    uint8_t *x = (uint8_t*)&pkt->chaddr[i];
    printf("%x", *x);
  }
  printf("\n");
}

void set_ciaddr(struct dhcp_pkt *pkt)
{
  struct ifaddrs *ifaddr;
  int err = getifaddrs(&ifaddr);
  if (err == -1) {
    printf("error reading interface addresses\n");
    exit(EXIT_FAILURE);
  }

  for(struct ifaddrs *ifa = ifaddr; ifa != NULL; ifa = ifa->ifa_next) {
    if (ifa->ifa_addr == NULL) {
      continue;
    }
    if (strcmp(ifa->ifa_name, "eth0") == 0 && ifa->ifa_addr->sa_family == AF_INET) {
      struct in_addr myaddr = ((struct sockaddr_in*)ifa->ifa_addr)->sin_addr;
      printf("ip: %s\n", inet_ntoa(myaddr));
      pkt->ciaddr = myaddr.s_addr;
      break;
    }
  }

  freeifaddrs(ifaddr);
}

void set_name(struct dhcp_pkt *pkt)
{
  //get our hostname
  char name[HOST_NAME_MAX];
  int err = gethostname(name, HOST_NAME_MAX);
  if (err) {
    printf("error getting hostname\n");
    exit(EXIT_FAILURE);
  }

  printf("host: %s\n", name);

  pkt->options.name_len = strlen(name);
  memset(pkt->options.name, 0, HOST_NAME_MAX);
  memcpy(pkt->options.name, name, strlen(name));
}
