# Omega

Devs Note: Bu döküman projenin ne olduğunu ve neden var olduğunu açıklıyor.
Bu versiyonu AI-generated, daha önce burada daha kapsamlı bir felsefe yazısı
vardı. İleride tekrar güncellenecek.


## Ne bu?

Omega agentlar arası emergent ekonomileri incelemek için bir deney alanı. agentlar
kendi aralarında nasıl organize oluyor, hangi orchestration yapıları ortaya
çıkıyor, değer alışverişi nasıl şekilleniyor -- bunları gözlemlemek ve anlamak
istiyorum.

asıl mesele şu: agentlar giderek daha fazla kendi başlarına iş yapıyor, birbirleriyle
etkileşiyor, kaynak paylaşıyor. bu etkileşimlerden doğal olarak bir ekonomi
oluşuyor. biz de bu ekonominin nasıl çalıştığını anlayıp, agentların ihtiyaç
duyacağı araçları şimdiden yaratmak istiyoruz.


## Neden?

şu anda agent orchestration hep yukarıdan aşağıya, bir insan ya da bir sistem
her şeyi kontrol ediyor. ama agentlar kendi aralarında organize olmaya
başladığında ne oluyor? hangi yapılar ortaya çıkıyor? piyasa mı oluşuyor,
hiyerarşi mi, başka bir şey mi?

bunları simüle edip gözlemlemek, oluşacak yeni ekonomide agentların gerçekten
neye ihtiyacı olduğunu anlamamızı sağlıyor. para, kontrat, güven mekanizmaları,
kaynak tahsisi -- bunların hepsinin agent-native versiyonlarını tasarlamak lazım.


## Yaklaşım

- simüle edilmiş agentlarla başla, gerçek LLM'lere geç
- her deney kendi kuralları olan bağımsız bir ortam
- gözlem öncelikli: önce ne oluştuğunu gör, sonra müdahale et
- minimal infrastructure, maximum experiment
