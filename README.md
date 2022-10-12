# Shlyuz Implant
This project is a fully featured Golang port of [Shlyuz](https://github.com/shlyuz/). For more information on Shlyuz, loosely modeled after Assassin as described in Vault 7, please [refer to my series of blog posts on it](https://und3rf10w.github.io/posts/2022/01/08/shlyuz-1-influences.html).

# Features
This implementation of a Shlyuz implant has a number of features that make it enticing:
- Asymmetric encryption of communications using [NaCl](https://nacl.cr.yp.to/box.html)
- Symmetric encryption of communications using [RC6](https://en.wikipedia.org/wiki/RC6)
- Cross platform compatability for desktops
    - Windows ✅
    - OSX ✅
    - Linux ✅
- Compiled binary (versus the python package of [the previously released implant PoC](https://github.com/shlyuz/mac_implant))
- Enhanced Execution Methods loosely modelled after the [ICE Standard as described in Valut 7](https://wikileaks.org/ciav7p1/cms/files/ICE-Spec-v3-final-SECRET.pdf) (⚠️ WARNING: Wikileaks link)

# Usage
1. Generate an implant configuration using the [teamserver setup application](https://github.com/shlyuz/teamserver/blob/master/setup.py)
2. Place the encrypted `shlyuz.conf` for the implant in `configs/`
3. (⚠️ Subject to change) Create a file `symkey` in `configs/`, with the contents being the generated config encryption key received from the teamserver setup application
4. Compile the implant
5. Distribute the compiled implant

# Donate
If you enjoyed this project, donations are accepted at und3rf10w.eth