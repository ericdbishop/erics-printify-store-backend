# Eric's Printify Store Backend

This is a generic version of the backend API server I wrote for [erichastourettes.com](https://erichastourettes.com) in Golang.
It manages
tasks such as user cart and session handling through an SQLite DB, as well as interfacing with the
APIs of Stripe (payment processor) and Printify (print-on-demand service) for
checkout and order creation functionality. 
It has been over a year since I launched the site and I wanted to share the work I put into creating a functional ecommerce platform on the backend.
I may put some effort into improving this code in the future.

I went through and cleaned up [erichastourettes.com](https://erichastourettes.com) specific references. 
If you plan to use any of the code, be sure to closely look at how you may need to modify it to fit your own needs.

The API exposes the following endpoints to the client:

`/api/items`

`/api/retrieve_cart`

`/api/add_to_cart`

`/api/remove_from_cart`

`/api/checkout`

It requires the following environment variables to be configured within your .env:

`PRINTIFY_API_TOKEN` Your Printify API token

`SHOP_ID` Your printify shop ID number

`STRIPE_SECRET` Your Stripe account secret token

`STRIPE_WEBHOOK_SECRET` Your Stripe webhook secret token

`LOGFILE` Log file name

`CSRF_AUTH_TOKEN` Random CSRF authorization token
