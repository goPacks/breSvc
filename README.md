Business Rules Engine (BRE)
---------------------------

Introduction
------------
Welcome to my home grown Business Rules Engine. This is my first attempt at using GO for backend development. I just competed finished 4 months (14th May 2021)  of GO training at Go School, facilitated by Nanyang Polytechnic, Singapore. In fact the works listed in this repository is my submission of an assignment which comes at end of the 4 months.

Problem Statement
-----------------
Why did I choose to develop a BRE for the assignment? The BRE I developed is generic and can be used for any industry. I spent over 20 years in the Retail Industry and I wanted to use the assignment to address a real problem in the retail industry. The problem being the disconnect between Retail Merchandisers and IT. 
Merchandisers want to drive sales and the best way they can reach the sales target is to create innovative promotion schemes. Unfortunately, most of their creative and innovative work is restricted by the frontend systems. The purpose of the BRE is to provide dynamic processing of the promotion schemes at the frontend system.

User Centric - Design Thinking
------------------------------
For many years, I have observed how merchandisers create promotion schemes. The primary tool they use is Excel where they load the data and slice and dice the data as they require. At the end they copy and paste the data into a host system to deliver the promotions to the front end. 
Excel is in the comfort zone of merchandisers. It is tool that are most comfortable with. So I decided to work within the comfort zone of the user and use Excel to create and test the rules.

Possible Designs Routes
-----------------------
At time of this writing, I have only seen one other BRE written in GO. I explored it and found a maze of code. I wanted something more that was like having a tracer to check which rules were processed and dynamic changing of the provided facts and addition of new facts whilst processing the rules. I considered the option to tweak the BRE tool.  However, I decided it would be a lot of work trying to tweak the code and there the learning outcome will not be achieved. And by trainers will be very upset as it wll not be 100% of my work.

How to Use
----------
Ok enough of my mumblings. 
You would want to know how to use the BRE which I have nick named breSvc

1. The breSvc works with MONG DB. So the first thing you need to do Is to download and install MONGO DB. 
You can use the following link. See https://www.mongodb.com/1. On completion you need to create database breSvc with a user colletion as shown below:

                        {"UserId":"hardeep","pswdhash":"999989"}


2. Next thing is to clone the repositry found here onto local machine. You can run it at the command prompt, by entring breSvc.exe

3. In the samples folder you can find 2 excel files. 
   (a) breSample.xlsm - Use to test 4 business rules
   (b) breStressTest.xlsm - Use to stress the service with 40 business rules and 10,0000 sku's.
   
Works in Progress
-----------------
There is more work to be done. I be adding on more samples and tryting different promotion schemes. At the moment the BRE works at item level for retail. I am working on next version at header and payment level.
 
